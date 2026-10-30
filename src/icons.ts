import {
  AppWindow, ArrowDownToLine, ArrowLeftRight, Battery, BrushCleaning,
  Check, ChevronDown, ChevronRight, Cpu, Folder, Globe, Gpu, HardDrive,
  Gauge, PanelTop, Laptop, List, MemoryStick, Pause, Play, RefreshCw,
  Search, Settings, X, Zap, Star, Copy, EthernetPort, Power, ArrowLeft, File, FolderOpen, ScanSearch, Trash2, createElement, type IconNode,
} from "lucide";

const icons: Record<string, IconNode> = {
  star: Star,
  copy: Copy,
  ports: EthernetPort,
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
