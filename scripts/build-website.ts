import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { zh } from "./website-copy";

const site = "https://rickyshin93.github.io/RockshinOps/";
const repo = "https://github.com/rickyshin93/RockshinOps";
const source = readFileSync("website/index.html", "utf8");
const escape = (value: string) => value.replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
const plain = (value: string) => value.replace(/<[^>]*>/g, "").replaceAll("&amp;", "&").replaceAll("&gt;", ">").replaceAll("&lt;", "<").trim();
const en = Object.fromEntries([...source.matchAll(/data-i18n="([^"]+)"[^>]*>([^<]*)</g)].map(match => [match[1]!, plain(match[2]!)]));
const titles = {
  en: "RockshinOps — Open-source system monitor for macOS, Windows & Linux",
  zh: "RockshinOps — macOS、Windows、Linux 免费开源系统监控工具",
};
const descriptions = {
  en: "Monitor CPU, memory, disk, network, processes and connections with RockshinOps. Free, open-source desktop monitoring with menu bar metrics and English/Chinese UI.",
  zh: "RockshinOps 免费开源桌面系统监控工具：查看 CPU、内存、磁盘、网络、进程和连接，支持菜单栏指标、中英文界面及 macOS、Windows、Linux。",
};

for (const language of ["en", "zh"] as const) {
  const chinese = language === "zh";
  const url = site + (chinese ? "zh/" : "");
  const copy: Record<string, string> = chinese ? zh : en;
  let html = source.replace(/\s*<!-- SEO:start -->[\s\S]*?<!-- SEO:end -->/, "");
  html = html.replace(/\s*<(?:meta (?:name="(?:description|robots|twitter:[^"]+)"|property="og:[^"]+")[^>]*|link rel="(?:canonical|alternate|describedby|sitemap)"[^>]*)>/g, "").replace(/\s*<title>[^<]*<\/title>/, "");
  const structured = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    "@id": site + "#application",
    name: "RockshinOps", url: site, description: descriptions[language],
    applicationCategory: "UtilitiesApplication", operatingSystem: "macOS, Windows, Linux",
    isAccessibleForFree: true, license: repo + "/blob/main/LICENSE",
    downloadUrl: repo + "/releases/latest", sameAs: [repo],
    inLanguage: ["en", "zh-CN"], screenshot: site + `assets/rockshinops-brand/overview-${language}.jpg`,
    offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
  };
  const metadata = `
  <!-- SEO:start -->
  <title>${escape(titles[language])}</title>
  <meta name="description" content="${escape(descriptions[language])}">
  <meta name="robots" content="index,follow,max-image-preview:large">
  <link rel="canonical" href="${url}">
  <link rel="alternate" hreflang="en" href="${site}">
  <link rel="alternate" hreflang="zh-CN" href="${site}zh/">
  <link rel="alternate" hreflang="x-default" href="${site}">
  <link rel="alternate" type="text/markdown" href="${url}index.md">
  <link rel="describedby" type="text/plain" href="${site}llms.txt">
  <link rel="sitemap" type="application/xml" href="${site}sitemap.xml">
  <meta property="og:type" content="website">
  <meta property="og:site_name" content="RockshinOps">
  <meta property="og:title" content="${escape(titles[language])}">
  <meta property="og:description" content="${escape(descriptions[language])}">
  <meta property="og:url" content="${url}">
  <meta property="og:locale" content="${chinese ? "zh_CN" : "en_US"}">
  <meta property="og:locale:alternate" content="${chinese ? "en_US" : "zh_CN"}">
  <meta property="og:image" content="${site}assets/rockshinops-brand/overview-${language}.jpg">
  <meta property="og:image:alt" content="${chinese ? "RockshinOps 概览界面，演示数据" : "RockshinOps overview with demonstration data"}">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="${escape(titles[language])}">
  <meta name="twitter:description" content="${escape(descriptions[language])}">
  <meta name="twitter:image" content="${site}assets/rockshinops-brand/overview-${language}.jpg">
  <script type="application/ld+json">${JSON.stringify(structured)}</script>
  <!-- SEO:end -->`;
  html = html.replace("</head>", metadata + "\n</head>");
  if (chinese) {
    html = html.replace('lang="en"', 'lang="zh-CN"').replace(/(data-i18n="([^"]+)"[^>]*>)[^<]*(<)/g, (_, before, key, after) => {
      if (!(key in zh)) throw new Error(`Missing Chinese website translation: ${key}`);
      return before + escape(copy[key]!) + after;
    });
    html = html.replaceAll('href="assets/', 'href="../assets/').replaceAll('src="assets/', 'src="../assets/').replace('href="styles.css"', 'href="../styles.css"').replace('src="main.js"', 'src="../main.js"');
    html = html.replace('href="zh/" hreflang="zh-CN"', 'href="../" hreflang="en"').replace('id="language-label">中文', 'id="language-label">English').replace('aria-label="切换为中文"', 'aria-label="Switch to English"');
    html = html.replace('aria-label="Main navigation"', 'aria-label="主导航"').replace('aria-label="Product principles"', 'aria-label="产品原则"').replace('aria-label="RockshinOps home"', 'aria-label="RockshinOps 首页"').replace('overview-en.jpg', 'overview-zh.jpg').replace('RockshinOps overview showing CPU, memory, GPU, disk, network and battery panels with demonstration data.', 'RockshinOps 概览，展示 CPU、内存、GPU、磁盘、网络与电池面板，使用演示数据。');
    html = html.replace('href="llms.txt"', 'href="../llms.txt"');
  }
  const directory = "website/" + (chinese ? "zh/" : "");
  mkdirSync(directory, { recursive: true });
  writeFileSync(directory + "index.html", html);
  const sections = [["monitorTitle", "monitorCopy"], ["inspectTitle", "inspectCopy"], ["spaceTitle", "spaceCopy"], ["quietTitle", "quietCopy"], ["downloadTitle", "downloadIntro"]].map(([title, body]) => `## ${copy[title!]}\n\n${copy[body!]}`);
  const downloads = [...html.matchAll(/class="download-link" href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/g)].map(match => `- [${plain(match[2]!).replace(/[↓↗]/g, "").trim()}](${match[1]})`);
  const faq = [1, 2, 3, 4].map(n => `### ${copy["faq" + n]}\n\n${copy["answer" + n]}`);
  writeFileSync(directory + "index.md", `# RockshinOps\n\n${copy.heroCopy}\n\n[${chinese ? "官网" : "Official website"}](${url}) · [GitHub](${repo}) · [MIT](${repo}/blob/main/LICENSE)\n\n${sections.join("\n\n")}\n\n${copy.version}\n\n${downloads.join("\n")}\n\n${copy.macNote}\n\n${copy.winNote}\n\n${copy.linuxNote}\n\n${copy.platformNote}\n\n${faq.join("\n\n")}\n`);
}
writeFileSync("website/sitemap.xml", `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n  <url><loc>${site}</loc></url>\n  <url><loc>${site}zh/</loc></url>\n</urlset>\n`);
writeFileSync("website/llms.txt", `# RockshinOps\n\n> Free, MIT-licensed desktop system monitor for macOS, Windows and Linux.\n\nRockshinOps reads native CPU, memory, disk, network, process and connection data. It provides persistent menu bar metrics, disk-space analysis and reviewed cleanup. History stays in memory for the current session; no cloud account is required. Menu-bar icon management and app-uninstall review are macOS features. Hardware telemetry and desktop acceptance vary by platform; see the product pages for limits.\n\n## Product\n\n- [English product facts and downloads](${site}index.md): Features, installation notes, platform limits and FAQs, generated from the visible website.\n- [中文产品说明与下载](${site}zh/index.md): 功能、安装要求、平台边界与常见问题。\n\n## Source and releases\n\n- [GitHub repository](${repo}): Source, README and MIT license.\n- [Latest release](${repo}/releases/latest): Published installers and release notes; repository main can contain unreleased changes.\n- [Platform support](${repo}/blob/main/docs/platforms.md): Collector sources and availability.\n- [Project status](${repo}/blob/main/STATUS.md): Verification boundaries and known issues.\n`);
