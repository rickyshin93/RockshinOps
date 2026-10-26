# LocalOps public website Goal plan

- Source: User request: modern technology-styled GitHub Pages promotional website, English default, Chinese support, publish with the app.
- Created: 2026-10-27
- Execution mode: autonomous
- Overall status: done

## 全局边界

- In scope: Static marketing page, authentic product UI with clearly labeled synthetic demo readings, bilingual copy, release links and GitHub Pages deployment.
- Out of scope: App features, analytics, accounts, new services and custom domains.
- Constraints: Preserve desktop code; no private device names, process data or paths in public assets. STATUS.md remains the progress authority.
- Assumptions: Deploy the existing public repository to its project URL at rickyshin93.github.io/local-ops/; the user authorized publication.

## 需求覆盖

| Requirement | Source | Goal | Acceptance |
|---|---|---|---|
| REQ-01 | User: modern technology-style promotional website | Goal 01 | Desktop/tablet/mobile rendering inspected; product imagery and working download links |
| REQ-02 | User: English default and Chinese | Goal 01 | Fresh URL opens English; toggle changes all copy and document language; language URL survives reload |
| REQ-03 | User: github.io and publish together | Goal 02 | Public HTTPS page renders and deployment succeeds; links point to released v0.2.0 packages |

## 自主预检

- Status: passed
- Command availability: Goal 01 uses installed Bun, Python and CUA browser tools. Goal 02 uses authenticated gh and SSH Git.
- External prerequisites: Public repository and v0.2.0 assets verified; Pages currently returns 404 and must be enabled.
- Authorization: Existing user instruction explicitly authorizes website creation and publication; app release already published.

## Goal 01 — Bilingual responsive product website

- Status: done
- Depends on: none

### 1. 目标

Explain LocalOps visually and let visitors choose their platform download.

### 2. 边界

- Allowed: website/, docs and a screenshot-only synthetic UI fixture under ignored work/.
- Excluded: Desktop functionality changes and fabricated performance claims.
- Impact: Standalone static marketing surface only.

### 3. 测试

- Commands: `bun run typecheck`; `bun run test:web`; `python3 -m http.server 4177 --directory website`.
- Cases: English default, Chinese toggle/reload, all release links, keyboard focus, reduced motion, desktop/tablet/mobile layout, console errors and asset loads.
- Regression: The existing app source is unchanged; previous native release gates remain valid.

### 4. 验收标准

- [x] Every visitor-facing section is available in English and Chinese; default English.
- [x] Responsive rendering and interactions verified in a browser with no horizontal overflow or website-origin console errors.
- [x] Download links match published assets; screenshot uses actual UI with labeled demo data and no private content.

### 5. 交付结果

- website/index.html, styles.css, main.js and locally hosted assets.
- Design decisions in DESIGN.md and verification evidence under work/logs/.

### 6. 执行记录

- Completed: 2026-10-27
- Commands run: bun run typecheck; bun run test:web; bun build website/main.js --target browser --outfile work/site-preview/main.bundle.js; local HTML/asset/release link audit.
- Results: 18 frontend tests pass, TypeScript and website JS pass. Both languages render at 360/768/1440px without horizontal overflow. Toggle/reload, FAQ keyboard, anchor navigation, pause and reduced motion verified. No website-origin console errors; unrelated installed-extension errors are excluded.
- Artifacts: website/; work/logs/website-verification.json; work/logs/website-desktop-en.jpg; work/logs/website-desktop-zh.jpg; work/logs/website-mobile-zh.jpg.
- Known issues: none
- Attempts: 2
- Last failure: Resolved Chinese title wrapping at 360px and an 8px decorative overflow at 768px.
- Next strategy: none; both issues verified fixed in the final viewport matrix.

## Goal 02 — Public GitHub Pages website

- Status: done
- Depends on: Goal 01

### 1. 目标

Make the approved website reachable on github.io and keep it deployable from the repository.

### 2. 边界

- Allowed: Pages workflow and repository Pages configuration, commit/push and deployment.
- Excluded: Other repositories, custom domains and changes to app release binaries.
- Impact: Public website for LocalOps only.

### 3. 测试

- Commands: `gh api repos/rickyshin93/local-ops/pages`; `gh run list --workflow pages.yml --limit 1`; `curl -I https://rickyshin93.github.io/local-ops/`.
- Cases: Successful Pages deployment, public English and Chinese rendering, assets under repository subpath and download destinations.
- Regression: Existing release remains published and unchanged.

### 4. 验收标准

- [x] Pages deployment succeeds and public HTTPS URL returns the website.
- [x] Online English/Chinese pages and key navigation/download links verified in browser.
- [x] README and STATUS reflect the actual delivered website.

### 5. 交付结果

- .github/workflows/pages.yml and public https://rickyshin93.github.io/local-ops/.
- Updated README files and STATUS.md.

### 6. 执行记录

- Completed: 2026-10-27
- Commands run: gh api repos/rickyshin93/local-ops/pages; gh run list --workflow pages.yml; HTTPS download and byte comparison of six public assets; native Chrome public-page inspection.
- Results: Pages run 40378884753 succeeded for 05b6f7d. Public HTML, CSS, JS, both product images and font return HTTP 200 and match local files byte-for-byte. Public English, Chinese toggle, download anchor and five installer destinations verified in Chrome. Balanced Chinese download-heading wrapping visually verified locally before deployment.
- Artifacts: https://rickyshin93.github.io/local-ops/; https://github.com/rickyshin93/local-ops/actions/runs/40378884753; work/logs/website-public-verification.json; work/logs/website-final-preview.png.
- Known issues: none
- Attempts: 2
- Last failure: Browser extension inspection timed out on the public origin; native Chrome accessibility and screenshots completed the public acceptance. An orphaned Chinese heading ending was fixed and redeployed.
- Next strategy: none; online acceptance complete.

## 全局验收

- [x] All goals completed with evidence.
- [x] Public website to release download path verified.
- [x] Relevant project checks passed.
- [x] Docs and project status synchronized.
