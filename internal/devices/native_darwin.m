#import <Cocoa/Cocoa.h>
#import <CoreWLAN/CoreWLAN.h>
#import <CoreLocation/CoreLocation.h>
#import <CoreBluetooth/CoreBluetooth.h>
#import <IOBluetooth/IOBluetooth.h>
#import <CoreAudio/CoreAudio.h>
#import <IOKit/IOKitLib.h>
#import <AudioToolbox/AudioHardwareService.h>

// C comparisons and CoreAudio Boolean values otherwise box as JSON numbers.
static NSNumber *JSONBool(BOOL value) { return value ? @YES : @NO; }

static void OnMain(dispatch_block_t block) {
 if ([NSThread isMainThread]) block(); else dispatch_sync(dispatch_get_main_queue(), block);
}
@interface RockshinDevicePermissions : NSObject <CLLocationManagerDelegate,CBCentralManagerDelegate>
@property(nonatomic,strong) CLLocationManager *location;
@property(nonatomic,strong) CBCentralManager *bluetooth;
@end
@implementation RockshinDevicePermissions
- (void)centralManagerDidUpdateState:(CBCentralManager *)manager {}
- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)manager {}
@end
static RockshinDevicePermissions *permissions;
static NSString *LocationStatus(void) {
 __block CLAuthorizationStatus status;
 BOOL enabled=CLLocationManager.locationServicesEnabled;
 OnMain(^{
  if(!permissions)permissions=[RockshinDevicePermissions new];
  if(!permissions.location){permissions.location=[CLLocationManager new];permissions.location.delegate=permissions;}
  status=permissions.location.authorizationStatus;
 });
 if (!enabled) return @"disabled";
 switch(status) {
  case kCLAuthorizationStatusNotDetermined:return @"notDetermined";
  case kCLAuthorizationStatusRestricted:return @"restricted";
  case kCLAuthorizationStatusDenied:return @"denied";
  default:return @"allowed";
 }
}
static NSString *BluetoothAuthorization(void) {
 switch(CBManager.authorization) {
  case CBManagerAuthorizationAllowedAlways:return @"allowed";
  case CBManagerAuthorizationDenied:return @"denied";
  case CBManagerAuthorizationRestricted:return @"restricted";
  default:return @"notDetermined";
 }
}
static NSString *Band(CWChannel *channel) {
 switch(channel.channelBand) {
  case kCWChannelBand2GHz:return @"2.4 GHz";
  case kCWChannelBand5GHz:return @"5 GHz";
  case kCWChannelBand6GHz:return @"6 GHz";
  default:return @"";
 }
}
static NSString *PHY(CWPHYMode mode) {
 switch(mode) {
  case kCWPHYMode11a:return @"802.11a";
  case kCWPHYMode11b:return @"802.11b";
  case kCWPHYMode11g:return @"802.11g";
  case kCWPHYMode11n:return @"Wi-Fi 4";
  case kCWPHYMode11ac:return @"Wi-Fi 5";
  case kCWPHYMode11ax:return @"Wi-Fi 6";
  default:return @"";
 }
}
static id Width(CWChannel *channel) {
 switch(channel.channelWidth) {
  case kCWChannelWidth20MHz:return @20;
  case kCWChannelWidth40MHz:return @40;
  case kCWChannelWidth80MHz:return @80;
  case kCWChannelWidth160MHz:return @160;
  default:return NSNull.null;
 }
}
static NSString *Security(CWSecurity security) {
 switch(security) {
  case kCWSecurityNone:return @"Open";
  case kCWSecurityWEP:return @"WEP";
  case kCWSecurityWPAPersonal:return @"WPA Personal";
  case kCWSecurityWPAPersonalMixed:return @"WPA/WPA2 Personal";
  case kCWSecurityWPA2Personal:return @"WPA2 Personal";
  case kCWSecurityWPAEnterprise:return @"WPA Enterprise";
  case kCWSecurityWPAEnterpriseMixed:return @"WPA/WPA2 Enterprise";
  case kCWSecurityWPA2Enterprise:return @"WPA2 Enterprise";
  case kCWSecurityWPA3Personal:return @"WPA3 Personal";
  case kCWSecurityWPA3Enterprise:return @"WPA3 Enterprise";
  case kCWSecurityWPA3Transition:return @"WPA2/WPA3 Personal";
  case kCWSecurityOWE:return @"OWE";
  case kCWSecurityOWETransition:return @"OWE Transition";
  case kCWSecurityDynamicWEP:return @"Dynamic WEP";
  default:return @"";
 }
}
static NSDictionary *ReadWiFi(void) {
 CWInterface *wifi=CWWiFiClient.sharedWiFiClient.interface;
 BOOL linked=wifi && wifi.powerOn && wifi.activePHYMode!=kCWPHYModeNone;
 CWChannel *channel=linked?wifi.wlanChannel:nil;
 NSInteger signal=linked?wifi.rssiValue:0, noise=linked?wifi.noiseMeasurement:0;
 // CoreWLAN returns zero on failure; this is not a measured zero-Mbps link.
 double rate=linked?wifi.transmitRate:0;
 return @{@"available":JSONBool(wifi!=nil),@"powered":JSONBool(wifi.powerOn),@"connected":JSONBool(linked),@"interface":wifi.interfaceName?:@"",@"ssid":linked?(wifi.ssid?:@""):@"",@"location":LocationStatus(),@"signal":signal!=0?@(signal):NSNull.null,@"noise":noise!=0?@(noise):NSNull.null,@"rate":rate>0?@(rate):NSNull.null,@"channel":channel?@(channel.channelNumber):NSNull.null,@"band":channel?Band(channel):@"",@"width":channel?Width(channel):NSNull.null,@"phy":linked?PHY(wifi.activePHYMode):@"",@"security":linked?Security(wifi.security):@"",@"source":@"CoreWLAN"};
}
// Only join battery readings by the exact device address; names are not identities.
static NSString *NormalizedAddress(NSString *address) {
 return [[address stringByReplacingOccurrencesOfString:@":" withString:@"-"] lowercaseString];
}
static NSDictionary *HIDBatteries(void) {
 NSMutableDictionary *batteries=[NSMutableDictionary dictionary];
 io_iterator_t iterator=IO_OBJECT_NULL;
 if(IOServiceGetMatchingServices(kIOMainPortDefault,IOServiceMatching("AppleDeviceManagementHIDEventService"),&iterator)!=KERN_SUCCESS)return batteries;
 io_object_t entry;
 while((entry=IOIteratorNext(iterator))) {
  id address=CFBridgingRelease(IORegistryEntryCreateCFProperty(entry,CFSTR("DeviceAddress"),kCFAllocatorDefault,0));
  id value=CFBridgingRelease(IORegistryEntryCreateCFProperty(entry,CFSTR("BatteryPercent"),kCFAllocatorDefault,0));
  if([address isKindOfClass:NSString.class] && [value isKindOfClass:NSNumber.class] && [value integerValue]>=0 && [value integerValue]<=100)batteries[NormalizedAddress(address)]=value;
  IOObjectRelease(entry);
 }
 IOObjectRelease(iterator);
 return batteries;
}
static NSDictionary *ReadBluetooth(void) {
 NSString *authorization=BluetoothAuthorization();
 NSMutableArray *devices=[NSMutableArray array];
 IOBluetoothHostController *controller=nil;
 if ([authorization isEqualToString:@"allowed"]) {
  controller=IOBluetoothHostController.defaultController;
  NSDictionary *batteries=HIDBatteries();
  for(IOBluetoothDevice *device in IOBluetoothDevice.pairedDevices) {
   NSString *kind=@"device";
   if(device.deviceClassMajor==kBluetoothDeviceClassMajorAudio)kind=@"audio";
   if(device.deviceClassMajor==kBluetoothDeviceClassMajorPeripheral)kind=@"peripheral";
   if(device.deviceClassMajor==kBluetoothDeviceClassMajorComputer)kind=@"computer";
   [devices addObject:@{@"id":device.addressString,@"name":device.name?:device.addressString,@"connected":JSONBool(device.isConnected),@"kind":kind,@"battery":batteries[NormalizedAddress(device.addressString)]?:NSNull.null}];
  }
 }
 [devices sortUsingComparator:^NSComparisonResult(NSDictionary *a,NSDictionary *b) {
  if(![a[@"connected"] isEqual:b[@"connected"]])return [a[@"connected"] boolValue]?NSOrderedAscending:NSOrderedDescending;
  return [a[@"name"] localizedCaseInsensitiveCompare:b[@"name"]];
 }];
 return @{@"authorization":authorization,@"available":JSONBool(controller!=nil),@"powered":JSONBool(controller && controller.powerState==kBluetoothHCIPowerStateON),@"devices":devices,@"source":@"IOBluetooth · IORegistry BatteryPercent"};
}
static AudioObjectPropertyAddress AudioAddress(AudioObjectPropertySelector selector,AudioObjectPropertyScope scope) {
 return (AudioObjectPropertyAddress){selector,scope,kAudioObjectPropertyElementMain};
}
static NSString *AudioString(AudioDeviceID device,AudioObjectPropertySelector selector) {
 AudioObjectPropertyAddress address=AudioAddress(selector,kAudioObjectPropertyScopeGlobal);
 CFStringRef value=NULL;UInt32 size=sizeof(value);
 if(AudioObjectGetPropertyData(device,&address,0,NULL,&size,&value)!=noErr)return nil;
 return CFBridgingRelease(value);
}
static AudioDeviceID DefaultOutput(OSStatus *status) {
 AudioDeviceID device=kAudioObjectUnknown;UInt32 size=sizeof(device);
 AudioObjectPropertyAddress address=AudioAddress(kAudioHardwarePropertyDefaultOutputDevice,kAudioObjectPropertyScopeGlobal);
 *status=AudioObjectGetPropertyData(kAudioObjectSystemObject,&address,0,NULL,&size,&device);
 return device;
}
static NSArray *Outputs(NSString **error) {
 AudioObjectPropertyAddress address=AudioAddress(kAudioHardwarePropertyDevices,kAudioObjectPropertyScopeGlobal);
 UInt32 size=0;OSStatus status=AudioObjectGetPropertyDataSize(kAudioObjectSystemObject,&address,0,NULL,&size);
 if(status!=noErr){*error=[NSString stringWithFormat:@"CoreAudio enumerate outputs: %d",(int)status];return @[];}
 NSMutableData *buffer=[NSMutableData dataWithLength:size];
 status=AudioObjectGetPropertyData(kAudioObjectSystemObject,&address,0,NULL,&size,buffer.mutableBytes);
 if(status!=noErr){*error=[NSString stringWithFormat:@"CoreAudio read outputs: %d",(int)status];return @[];}
 NSMutableArray *outputs=[NSMutableArray array];AudioDeviceID *ids=buffer.mutableBytes;
 for(UInt32 i=0;i<size/sizeof(AudioDeviceID);i++) {
  AudioObjectPropertyAddress streams=AudioAddress(kAudioDevicePropertyStreamConfiguration,kAudioDevicePropertyScopeOutput);
  UInt32 streamSize=0;
  status=AudioObjectGetPropertyDataSize(ids[i],&streams,0,NULL,&streamSize);
  if(status!=noErr || !streamSize)continue;
  NSMutableData *streamData=[NSMutableData dataWithLength:streamSize];
  status=AudioObjectGetPropertyData(ids[i],&streams,0,NULL,&streamSize,streamData.mutableBytes);
  if(status!=noErr)continue; // Device may disappear between enumeration and inspection.
  AudioBufferList *list=streamData.mutableBytes;UInt32 channels=0;
  for(UInt32 n=0;n<list->mNumberBuffers;n++)channels+=list->mBuffers[n].mNumberChannels;
  if(!channels)continue;
  NSString *uid=AudioString(ids[i],kAudioDevicePropertyDeviceUID);
  if(!uid)continue;
  [outputs addObject:@{@"id":uid,@"name":AudioString(ids[i],kAudioObjectPropertyName)?:uid,@"nativeId":@(ids[i])}];
 }
 return outputs;
}
static NSDictionary *ReadAudio(void) {
 NSString *error=@"";NSArray *outputs=Outputs(&error);
 OSStatus status=noErr;AudioDeviceID device=DefaultOutput(&status);
 NSString *uid=device!=kAudioObjectUnknown?AudioString(device,kAudioDevicePropertyDeviceUID):@"";
 if(status!=noErr)error=[NSString stringWithFormat:@"CoreAudio default output: %d",(int)status];
 id volume=NSNull.null;Boolean settable=false;
 AudioObjectPropertyAddress address=AudioAddress(kAudioHardwareServiceDeviceProperty_VirtualMainVolume,kAudioDevicePropertyScopeOutput);
 if(device!=kAudioObjectUnknown && AudioObjectHasProperty(device,&address)) {
  Float32 value=0;UInt32 size=sizeof(value);
  status=AudioObjectGetPropertyData(device,&address,0,NULL,&size,&value);
  if(status==noErr)volume=@(value*100);else error=[NSString stringWithFormat:@"CoreAudio volume: %d",(int)status];
  if(AudioObjectIsPropertySettable(device,&address,&settable)!=noErr)settable=false;
 }
 return @{@"outputs":outputs,@"defaultId":uid?:@"",@"volume":volume,@"volumeSettable":JSONBool(settable),@"error":error,@"source":@"CoreAudio"};
}
static NSDictionary *ReadState(void) {
 return @{@"supported":@YES,@"wifi":ReadWiFi(),@"bluetooth":ReadBluetooth(),@"audio":ReadAudio()};
}
static NSString *Perform(NSDictionary *request,NSMutableDictionary *result) {
 NSString *action=request[@"action"],*target=request[@"id"];
 if([action isEqualToString:@"read"]) { result[@"state"]=ReadState(); return nil; }
 if([action isEqualToString:@"location"] || [action isEqualToString:@"bluetoothPermission"]) {
  BOOL location=[action isEqualToString:@"location"];
  NSString *status=location?LocationStatus():BluetoothAuthorization();
  if(![status isEqualToString:@"notDetermined"])return @"Permission has already been decided; use System Settings to change it.";
  OnMain(^{
   if(!permissions)permissions=[RockshinDevicePermissions new];
   if(location){permissions.location=[CLLocationManager new];permissions.location.delegate=permissions;[permissions.location requestWhenInUseAuthorization];}
   else permissions.bluetooth=[[CBCentralManager alloc] initWithDelegate:permissions queue:dispatch_get_main_queue() options:@{CBCentralManagerOptionShowPowerAlertKey:@NO}];
  });
  return nil;
 }
 if([action isEqualToString:@"scan"]) {
  if(![LocationStatus() isEqualToString:@"allowed"])return @"Location permission is required to scan Wi-Fi network names.";
  CWInterface *wifi=CWWiFiClient.sharedWiFiClient.interface;
  if(!wifi || !wifi.powerOn)return @"Wi-Fi is unavailable or powered off.";
  NSError *error=nil;NSSet<CWNetwork *> *networks=[wifi scanForNetworksWithSSID:nil error:&error];
  if(error)return error.localizedDescription;
  NSMutableDictionary *unique=[NSMutableDictionary dictionary];
  for(CWNetwork *network in networks) {
   if(!network.ssid.length)continue;
   NSString *band=Band(network.wlanChannel);
   NSString *key=[NSString stringWithFormat:@"%@\n%@",network.ssid,band];
   if(unique[key] && [unique[key][@"signal"] integerValue]>=network.rssiValue)continue;
   unique[key]=@{@"ssid":network.ssid,@"signal":@(network.rssiValue),@"channel":@(network.wlanChannel.channelNumber),@"band":band,@"secure":JSONBool(![network supportsSecurity:kCWSecurityNone])};
  }
  result[@"networks"]=[unique.allValues sortedArrayUsingComparator:^NSComparisonResult(NSDictionary *a,NSDictionary *b){return [b[@"signal"] compare:a[@"signal"]];}];
  return nil;
 }
 if([action isEqualToString:@"connect"] || [action isEqualToString:@"disconnect"]) {
  if(![BluetoothAuthorization() isEqualToString:@"allowed"])return @"Bluetooth permission is required.";
  if(IOBluetoothHostController.defaultController.powerState!=kBluetoothHCIPowerStateON)return @"Bluetooth is powered off.";
  IOBluetoothDevice *found=nil;
  for(IOBluetoothDevice *device in IOBluetoothDevice.pairedDevices)if([device.addressString isEqualToString:target]){found=device;break;}
  if(!found)return @"The Bluetooth device is no longer paired.";
  BOOL connect=[action isEqualToString:@"connect"];
  if(found.isConnected==connect)return nil;
  IOReturn status=connect?[found openConnection:nil]:[found closeConnection];
  if(status!=kIOReturnSuccess)return [NSString stringWithFormat:@"Bluetooth %@ failed (%d).",action,(int)status];
  return nil; // The next snapshot reports the actual connection state, including pending OS transitions.
 }
 if([action isEqualToString:@"output"] || [action isEqualToString:@"volume"]) {
  NSString *error=@"";NSArray *outputs=Outputs(&error);if(error.length)return error;
  AudioDeviceID device=kAudioObjectUnknown;
  for(NSDictionary *output in outputs)if([output[@"id"] isEqualToString:target]){device=[output[@"nativeId"] unsignedIntValue];break;}
  if(device==kAudioObjectUnknown)return @"The audio output is no longer available.";
  OSStatus status=noErr;
  if([action isEqualToString:@"output"]) {
   AudioObjectPropertyAddress address=AudioAddress(kAudioHardwarePropertyDefaultOutputDevice,kAudioObjectPropertyScopeGlobal);
   status=AudioObjectSetPropertyData(kAudioObjectSystemObject,&address,0,NULL,sizeof(device),&device);
  }else{
   AudioDeviceID current=DefaultOutput(&status);
   if(status!=noErr || current!=device)return @"The default audio output changed. Refresh before adjusting volume.";
   AudioObjectPropertyAddress address=AudioAddress(kAudioHardwareServiceDeviceProperty_VirtualMainVolume,kAudioDevicePropertyScopeOutput);
   Boolean settable=false;
   status=AudioObjectIsPropertySettable(device,&address,&settable);
   if(status!=noErr || !settable)return @"This output does not support software volume control.";
   Float32 value=[request[@"value"] doubleValue]/100.0;
   status=AudioObjectSetPropertyData(device,&address,0,NULL,sizeof(value),&value);
  }
  if(status!=noErr)return [NSString stringWithFormat:@"CoreAudio %@ failed (%d).",action,(int)status];
  return nil;
 }
 return @"Unknown device action.";
}
char *rockshin_devices_request(const char *input) {
 @autoreleasepool {
  NSError *error=nil;
  NSDictionary *request=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:input] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:&error];
  NSMutableDictionary *result=[NSMutableDictionary dictionary];
  NSString *failure=error?error.localizedDescription:Perform(request,result);
  if(failure)result[@"error"]=failure;
  NSData *data=[NSJSONSerialization dataWithJSONObject:result options:0 error:&error];
  if(!data)return strdup("{\"error\":\"Could not encode native device response\"}");
  return strdup([[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
 }
}
