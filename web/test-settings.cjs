const puppeteer = require('puppeteer');

(async () => {
  console.log('Launching browser...');
  const browser = await puppeteer.launch({ headless: "new", args: ['--no-sandbox'] });
  const page = await browser.newPage();
  
  page.on('console', msg => console.log('BROWSER CONSOLE:', msg.text()));
  page.on('pageerror', err => console.log('BROWSER ERROR:', err.toString()));
  
  await page.goto('http://localhost:5173');
  await page.waitForTimeout(2000);
  
  console.log('App loaded. Clicking settings...');
  const handles = await page.$$('button');
  let clicked = false;
  for (let h of handles) {
      const target = await page.evaluate(el => el.getAttribute('data-panel-target'), h);
      if (target === 'settings') {
          await h.click();
          clicked = true;
          console.log('Clicked settings sidebar button!');
          break;
      }
  }
  
  if (!clicked) {
     console.log('Settings button not found via data-panel-target');
  }

  await page.waitForTimeout(1000);
  
  console.log('Checking DOM for Workstation Settings title...');
  const h2Text = await page.evaluate(() => {
     const h2s = Array.from(document.querySelectorAll('h2'));
     return h2s.map(h => h.textContent);
  });
  console.log('H2s found:', h2Text);

  console.log('Checking DOM for z-modal class...');
  const modalDiv = await page.evaluate(() => {
     const el = document.querySelector('.z-modal');
     if (el) {
         const style = window.getComputedStyle(el);
         return { found: true, zIndex: style.zIndex, display: style.display };
     }
     return { found: false };
  });
  console.log('Modal div status:', modalDiv);

  await browser.close();
})();
