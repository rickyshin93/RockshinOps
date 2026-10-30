const zh = document.documentElement.lang === 'zh-CN';
// Preserve existing shared Chinese links while serving crawlable static pages.
if (!zh && new URLSearchParams(location.search).get('lang') === 'zh') {
  location.replace(new URL(`zh/${location.hash}`, location.href).href);
}

const labels = zh
  ? { mac: '下载 Mac 版', windows: '下载 Windows 版', linux: '下载 Linux 版' }
  : { mac: 'Download for Mac', windows: 'Download for Windows', linux: 'Download for Linux' };
const names = { 'mac-arm64': 'macOS · Apple Silicon', 'mac-x64': 'macOS · Intel', 'windows-x64': 'Windows · x86_64', 'linux-deb': 'Linux · .deb', 'linux-tar': 'Linux · .tar.gz' };

// Browsers report Intel in every Mac user agent; ask Chromium for the architecture, otherwise infer it from the GPU.
async function macIsIntel() {
  try {
    const { architecture } = await navigator.userAgentData.getHighEntropyValues(['architecture']);
    if (architecture) return architecture === 'x86';
  } catch {}
  try {
    const gl = document.createElement('canvas').getContext('webgl');
    const info = gl.getExtension('WEBGL_debug_renderer_info');
    return /Intel|AMD|Radeon|NVIDIA/i.test(gl.getParameter(info ? info.UNMASKED_RENDERER_WEBGL : gl.RENDERER));
  } catch {
    return false;
  }
}

async function detectTarget() {
  const ua = navigator.userAgent;
  if (/Android|iPhone|iPad|iPod/.test(ua) || navigator.userAgentData?.mobile || (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)) return undefined;
  if (/Windows/.test(ua)) return 'windows-x64';
  if (/Macintosh/.test(ua)) return (await macIsIntel()) ? 'mac-x64' : 'mac-arm64';
  if (/Linux/.test(ua) && !/aarch64|arm/i.test(ua)) return /Ubuntu|Debian/.test(ua) ? 'linux-deb' : 'linux-tar';
  return undefined;
}

// Unrecognized visitors get the Apple Silicon build; the hint links to every other version.
detectTarget().then((target = 'mac-arm64') => {
  const link = document.querySelector(`.download-link[data-target="${target}"]`);
  const button = document.querySelector('#hero-download');
  button.href = link.href;
  button.textContent = labels[target.split('-')[0]];
  document.querySelector('#detected-label').textContent = names[target];
  document.querySelector('#detected').hidden = false;
});
