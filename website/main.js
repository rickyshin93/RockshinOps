const zh = {
  skip:'跳到正文', navFeatures:'功能', navDownload:'下载', release:'V0.2.1 已发布', releaseNote:'重新认识你的电脑', hero1:'电脑的每一刻，', hero2:'一目了然。', heroCopy:'从忙碌的 CPU，到逐渐装满的磁盘。在一个安静的界面里，看清电脑正在发生什么。', getApp:'下载 LocalOps', exploreSource:'探索源代码', free:'免费 · 开源', consoleLabel:'一眼看懂，心中有数', caption:'真实 macOS 界面 · 演示数据', pauseMotion:'暂停动效', principle1:'原生系统数据', principle2:'无需注册账户', principle3:'只读，安心查看', principle4:'中文与英文', featuresEyebrow:'01 / 看见表象之下', featuresTitle1:'少一点猜测，', featuresTitle2:'多一点了解。', featuresIntro:'保留有用的细节，去掉多余的干扰。从一个读数，深入理解它背后的系统活动。', monitorLabel:'观察', monitorTitle:'看清整个系统。', monitorCopy:'CPU、内存、磁盘与网络，一处掌握。查看每个核心的活动，回顾本次运行中最近一小时的变化。', monitorFoot:'一个概览，六个硬件面板。', inspectLabel:'理解', inspectTitle:'找到忙碌的源头。', inspectCopy:'查看进程，按应用归集资源占用，检查当前连接。顺着资源消耗，找到对应的活动。', processTag:'进程', appsTag:'应用', spaceLabel:'读懂磁盘空间', spaceTitle:'知道空间去了哪里。', spaceCopy:'查看缓存、日志和下载目录的大小。用只读分析理解磁盘占用，文件始终保持原样。', quietEyebrow:'需要时在场，其余时候安静', quietTitle:'关上窗口，仍然心中有数。', quietCopy:'把关心的指标留在菜单栏。隐藏窗口后，原生采样继续运行；再次打开时，变化尽在眼前。', allFeatures:'查看全部功能', downloadEyebrow:'02 / 开始看清你的电脑', downloadTitle:'为你的电脑，准备好了。', downloadIntro:'选择你的平台。无需账户、订阅，也不用配置云端控制台。', macDesc:'支持 Apple Silicon 与 Intel', macNote:'采用临时签名，尚未公证。', winDesc:'适用于 x86_64 电脑', winDownload:'下载安装程序', winNote:'安装包未签名，需要 WebView2。', linuxDesc:'适合你习惯的桌面', linuxArchive:'Linux 压缩包', linuxNote:'需要 GTK 3、WebKitGTK 4.1 和桌面托盘支持。', version:'当前版本 — V0.2.1', releaseDetails:'发行说明与安装指引', checksums:'SHA-256 校验清单', platformNote:'核心采集已在四个构建目标上测试。桌面交互已在 macOS Apple Silicon 验证；Windows/Linux 桌面交互及电池、GPU 实机覆盖仍在进行。GPU 与功耗读数取决于硬件和驱动支持。', faqEyebrow:'再了解一点', faqTitle:'你可能想知道。', faq1:'需要云端账户吗？', answer1:'不需要。LocalOps 在本机读取系统指标，历史数据仅保留在本次运行的内存中。无需账户，也无需配置托管监控服务。', faq2:'空间分析会删除文件吗？', answer2:'不会。分析是只读的，LocalOps 不会删除文件、卸载应用或结束进程。统计结果是文件逻辑大小，并不代表一定可回收的空间。', faq3:'应用支持中文吗？', answer3:'支持。应用默认跟随系统语言。也可以在设置中选择“跟随系统”“中文”或“English”，保存后持久生效。', faq4:'可以查看源代码吗？', answer4:'可以。LocalOps 采用 MIT 许可证，使用 Go、MyGo 和 TypeScript 构建。', readCode:'在 GitHub 查看代码 ↗', footerLine:'你的电脑，多一分了解。', license:'MIT 许可证',
};
const nodes = [...document.querySelectorAll('[data-i18n]')];
const en = Object.fromEntries(nodes.map(node => [node.dataset.i18n, node.textContent]));
const languageButton = document.querySelector('#language');
const motionButton = document.querySelector('#motion');
const image = document.querySelector('#product-image');
let language = 'en';
let paused = false;
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
function updateMotionLabel() {
  motionButton.textContent = reducedMotion.matches ? (language === 'zh' ? '已减少动态效果' : 'Reduced motion enabled') : paused ? (language === 'zh' ? '播放动效' : 'Play animation') : (language === 'zh' ? zh.pauseMotion : en.pauseMotion);
  motionButton.setAttribute('aria-pressed', String(paused || reducedMotion.matches));
  motionButton.disabled = reducedMotion.matches;
}
function renderLanguage() {
  language = new URLSearchParams(location.search).get('lang') === 'zh' ? 'zh' : 'en';
  const copy = language === 'zh' ? zh : en;
  for (const node of nodes) node.textContent = copy[node.dataset.i18n];
  document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
  document.title = language === 'zh' ? 'LocalOps — 电脑的每一刻，一目了然。' : 'LocalOps — Your machine. In the clear.';
  document.querySelector('meta[name="description"]').content = language === 'zh' ? '看清你的电脑。LocalOps 是适用于 macOS、Windows 和 Linux 的开源桌面系统监控工具。' : 'A clear view of your computer. LocalOps is an open-source desktop monitor for macOS, Windows and Linux.';
  document.querySelector('#language-label').textContent = language === 'zh' ? 'English' : '中文';
  languageButton.setAttribute('aria-label', language === 'zh' ? 'Switch to English' : '切换为中文');
  document.querySelector('nav').setAttribute('aria-label', language === 'zh' ? '主导航' : 'Main navigation');
  document.querySelector('.principles').setAttribute('aria-label', language === 'zh' ? '产品原则' : 'Product principles');
  document.querySelector('.header .brand').setAttribute('aria-label', language === 'zh' ? 'LocalOps 首页' : 'LocalOps home');
  image.src = `assets/localops-brand/overview-${language}.jpg`;
  image.alt = language === 'zh' ? 'LocalOps 概览，展示 CPU、内存、GPU、磁盘、网络与电池面板，使用演示数据。' : 'LocalOps overview showing CPU, memory, GPU, disk, network and battery panels with demonstration data.';
  for (const link of document.querySelectorAll('.brand')) link.href = language === 'zh' ? './?lang=zh' : './';
  updateMotionLabel();
}
languageButton.addEventListener('click', () => {
  const url = new URL(location.href);
  if (language === 'en') url.searchParams.set('lang', 'zh'); else url.searchParams.delete('lang');
  history.replaceState(null, '', url);
  renderLanguage();
});
motionButton.addEventListener('click', () => {
  paused = !paused;
  document.body.classList.toggle('motion-paused', paused);
  updateMotionLabel();
});
reducedMotion.addEventListener('change', updateMotionLabel);
window.addEventListener('popstate', renderLanguage);
renderLanguage();
