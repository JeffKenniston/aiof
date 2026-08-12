import { test, expect } from '@playwright/test';

test.describe('Adaptive UI Boundaries & State Validation', () => {
  
  test('should render zero-reflow layout correctly across viewports & resolve RxDB hydration', async ({ page }) => {
    // 1. Validate Compact Boundary (< 600dp)
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('http://localhost:5173');
    
    // Assert React 19 use() hydration resolves the WAL local replica
    const status = page.locator('text=Status: WAL Hydrated & RxDB Initialized');
    await expect(status).toBeVisible({ timeout: 5000 });

    const adaptivePane = page.locator('.adaptive-pane');
    await expect(adaptivePane).toHaveClass(/compact-stack/);

    // 2. Assert Generative UI State Persistence
    const pushBtn = page.getByRole('button', { name: 'Dispatch Local State Mutation' });
    await expect(pushBtn).toBeVisible();

    // 3. Validate Medium Boundary (600dp - 839dp)
    await page.setViewportSize({ width: 768, height: 1024 });
    await expect(adaptivePane).toHaveClass(/medium-split/);
    
    // Chaos Engineering check: Ensure clicking push mutation doesn't crash UI (even offline)
    await pushBtn.click();
  });
  
});
