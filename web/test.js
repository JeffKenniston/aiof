const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.launch({ headless: "new", args: ['--no-sandbox'] });
  const page = await browser.newPage();
  page.on('console', msg => console.log('BROWSER CONSOLE:', msg.text()));
  page.on('pageerror', err => console.log('BROWSER ERROR:', err.toString()));
  await page.goto('http://localhost:5173');
  // Wait for the app to load
  await page.waitForTimeout(2000);
  console.log('App loaded. Clicking settings...');
  // Click the settings button in sidebar
  // It has data-panel-target="settings" or contains text "Settings"
  const handles = await page.$$('button');
  for (let h of handles) {
      const text = await page.evaluate(el => el.textContent, h);
      if (text && text.includes('Settings')) {
          await h.click();
          console.log('Clicked settings!');
          break;
      }
  }
  await page.waitForTimeout(2000);
  console.log('Done.');
  await browser.close();
})();
