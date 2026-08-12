const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('http://localhost:5173/');
  await page.waitForLoadState('networkidle');
  await page.screenshot({ path: 'artifacts/screenshot.png' });
  const html = await page.content();
  console.log(html.substring(0, 500));
  const agents = ['DevOps', 'SecOps', 'DB Admin', 'UI Designer', 'Tech Writer', 'SRE (eBPF)', 'Chaos Monkey', 'MLOps', 'Product Owner', 'Localization (i18n)', 'QA Tester', 'Architect', 'Researcher', 'Validator', 'CodeDeveloper'];
  let foundCount = 0;
  for (const agent of agents) {
    if (html.includes(agent)) {
      foundCount++;
      console.log(`Found agent: ${agent}`);
    } else {
      console.log(`Missing agent: ${agent}`);
    }
  }
  console.log(`Total agents found: ${foundCount}/15`);
  await browser.close();
})();
