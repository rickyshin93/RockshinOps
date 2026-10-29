import {
  AppWindow, ArrowDownToLine, ArrowLeftRight, Battery, BrushCleaning,
  Check, ChevronDown, ChevronRight, Cpu, Folder, Globe, Gpu, HardDrive,
  Gauge, PanelTop, Laptop, List, MemoryStick, Pause, Play, RefreshCw,
  Search, Settings, X, Zap, createElement, type IconNode,
} from "lucide";

const icons: Record<string, IconNode> = {
  settings: Settings,
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
  folder: Folder,
};

export function icon(name: string): string {
  return createElement(icons[name] || Laptop, {
    "stroke-width": 1.75,
    "aria-hidden": "true",
  }).outerHTML;
}
