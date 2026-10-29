const language = document.documentElement.lang === 'zh-CN' ? 'zh' : 'en';
// Preserve existing shared Chinese links while serving crawlable static pages.
if (language === 'en' && new URLSearchParams(location.search).get('lang') === 'zh') {
  location.replace(new URL(`zh/${location.hash}`, location.href).href);
}
const motionButton = document.querySelector('#motion');
const pauseLabel = motionButton.textContent;
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
let paused = false;
function updateMotionLabel() {
  motionButton.textContent = reducedMotion.matches ? (language === 'zh' ? '已减少动态效果' : 'Reduced motion enabled') : paused ? (language === 'zh' ? '播放动效' : 'Play animation') : pauseLabel;
  motionButton.setAttribute('aria-pressed', String(paused || reducedMotion.matches));
  motionButton.disabled = reducedMotion.matches;
}
motionButton.addEventListener('click', () => {
  paused = !paused;
  document.body.classList.toggle('motion-paused', paused);
  updateMotionLabel();
});
reducedMotion.addEventListener('change', updateMotionLabel);
updateMotionLabel();
