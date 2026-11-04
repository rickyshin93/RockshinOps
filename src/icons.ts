import {
  Bluetooth, Headphones, Volume2, Mouse,
  AppWindow, ArrowDownToLine, ArrowLeftRight, Battery, BrushCleaning,
  Check, ChevronDown, ChevronRight, Cpu, Folder, Globe, Gpu, HardDrive,
  Gauge, PanelTop, Laptop, List, MemoryStick, Pause, Play, RefreshCw,
  Search, Settings, Thermometer, Fan, X, Zap, Star, Copy, EthernetPort, Power, ArrowLeft, File, FolderOpen, ScanSearch, Trash2, createElement, type IconNode,
} from "lucide";

const wifi: IconNode = [
  ["path", { d: "M1.538 7.895A16 16 0 0 1 22.462 7.895Q22.725 8.126 22.49 8.386L20.814 10.241Q20.58 10.501 20.317 10.27A12.8 12.8 0 0 0 3.683 10.27Q3.42 10.501 3.186 10.241L1.51 8.386Q1.275 8.126 1.538 7.895Z", fill: "currentColor", stroke: "none" }],
  ["path", { d: "M5.216 11.967A10.514 10.514 0 0 1 18.784 11.967Q19.047 12.198 18.813 12.457L17.137 14.313Q16.902 14.572 16.637 14.344A7.314 7.314 0 0 0 7.363 14.344Q7.098 14.572 6.863 14.313L5.187 12.457Q4.953 12.198 5.216 11.967Z", fill: "currentColor", stroke: "none" }],
  ["path", { d: "M9.205 16.383A4.571 4.571 0 0 1 14.795 16.383Q15.064 16.608 14.829 16.868L12.27 19.773Q12 20 11.73 19.773L9.171 16.868Q8.936 16.608 9.205 16.383Z", fill: "currentColor", stroke: "none" }],
];

const icons: Record<string, IconNode> = {
  devices: wifi,
  wifi,
  bluetooth: Bluetooth,
  headphones: Headphones,
  speaker: Volume2,
  mouse: Mouse,
  star: Star,
  copy: Copy,
  ports: EthernetPort,
  settings: Settings,
  launcher: Search,
  menubar: PanelTop,
  down: ChevronDown,
  check: Check,
  overview: Gauge,
  processes: List,
  connections: ArrowLeftRight,
  cpu: Cpu,
  memory: MemoryStick,
  gpu: Gpu,
  disk: HardDrive,
  network: Globe,
  energy: Zap,
  sensors: Thermometer,
  fan: Fan,
  battery: Battery,
  apps: AppWindow,
  cleanup: BrushCleaning,
  about: Laptop,
  refresh: RefreshCw,
  pause: Pause,
  play: Play,
  chevron: ChevronRight,
  hide: ArrowDownToLine,
  search: Search,
  close: X,
  power: Power,
  folder: Folder,
  back: ArrowLeft,
  file: File,
  reveal: FolderOpen,
  scan: ScanSearch,
  trash: Trash2,
};

const markup = new Map<IconNode, string>();

export function icon(name: string): string {
  const node = icons[name] || Laptop;
  const cached = markup.get(node);
  if (cached) return cached;
  const svg = createElement(node, {
    "stroke-width": 1.75,
    "aria-hidden": "true",
  }).outerHTML;
  markup.set(node, svg);
  return svg;
}

export function appIcon(path: string, native: boolean): string {
  if (!native) return `<span class="app-icon app-letter">${icon("apps")}</span>`;
  return `<img class="app-icon" src="appicon://localhost/?path=${encodeURIComponent(path)}" alt="">`;
}

export function processAppIcon(path: string, native: boolean): string {
  return path ? appIcon(path, native) : `<span class="app-icon process-system-icon">${icon("settings")}</span>`;
}
