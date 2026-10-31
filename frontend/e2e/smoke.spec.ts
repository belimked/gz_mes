import { expect, test, type Page } from '@playwright/test'

// Runs against the demo build by default, or a real stack when E2E_BASE_URL is set
// (then E2E_USER / E2E_PASSWORD must be an admin account).
const user = process.env.E2E_USER ?? 'admin'
const password = process.env.E2E_PASSWORD ?? 'admin123'

async function login(page: Page, u = user, p = password) {
  await page.goto('/')
  await expect(page).toHaveURL(/\/login/)
  await page.getByTestId('username').fill(u)
  await page.getByTestId('password').fill(p)
  await page.getByTestId('login-submit').click()
}

test('login, create and confirm a project, logout', async ({ page }) => {
  await login(page)
  await expect(page.getByTestId('side-menu')).toBeVisible()

  await page.getByTestId('side-menu').getByText('销售管理').click()
  await page.getByTestId('side-menu').getByText('项目管理').click()
  await expect(page).toHaveURL(/\/sale\/projects/)

  const code = `E2E-${Date.now()}`
  await page.getByTestId('project-new').click()
  await page.getByTestId('form-code').fill(code)
  await page.getByTestId('form-fullName').fill('端到端测试项目')
  await page.getByRole('button', { name: /保\s*存/ }).click()
  await expect(page.getByTestId('project-table')).toContainText(code)

  await page.getByTestId(`confirm-${code}`).click()
  await page.getByRole('button', { name: /确\s*定|OK/ }).click()
  const row = page.getByTestId('project-table').locator('tr', { hasText: code })
  await expect(row).toContainText('已确认')

  await page.getByTestId('user-menu').hover()
  await page.getByTestId('logout').click()
  await expect(page).toHaveURL(/\/login/)
})

test('scan records page lists seeded scans and filters by keyword', async ({ page }) => {
  await login(page)
  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('扫描记录').click()
  await expect(page).toHaveURL(/\/report\/scans/)
  const table = page.getByTestId('scan-table')
  await expect(table.locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  const barCode = (await table.locator('tbody tr.ant-table-row').first().locator('td').nth(3).innerText()).trim()
  await page.getByPlaceholder('条码 / 批次 / 工程 / 编号').fill(barCode)
  await page.getByRole('button', { name: /查\s*询/ }).click()
  const codeCells = table.locator('tbody tr.ant-table-row td:nth-child(4)')
  await expect(codeCells.filter({ hasNotText: barCode })).toHaveCount(0)
  await expect(codeCells).not.toHaveCount(0)
  await expect(codeCells).not.toHaveCount(20)
})

test('production balance page and progress summary toggle', async ({ page }) => {
  await login(page)
  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('生产平衡表').click()
  await expect(page).toHaveURL(/\/report\/production-balance/)
  await expect(page.getByTestId('balance-table').locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  await expect(page.getByTestId('balance-summary')).toContainText('批次')

  await page.getByTestId('side-menu').getByText('工程进度').click()
  await expect(page).toHaveURL(/\/report\/batch-progress/)
  await page.getByTestId('progress-view').getByText('工程汇总').click()
  await expect(page.getByTestId('report-table').locator('thead')).toContainText('完成率')
  await expect(page.getByTestId('export-csv')).toBeVisible()
})

test('wrong password shows an error', async ({ page }) => {
  await login(page, user, 'definitely-wrong')
  await expect(page.getByText('用户名或密码错误')).toBeVisible()
  await expect(page).toHaveURL(/\/login/)
})

test('every login style renders a working form', async ({ page }) => {
  for (const style of ['aurora', 'nebula', 'tide', 'ember']) {
    await page.goto(`/login?style=${style}`)
    await expect(page.getByTestId('login-style-switcher')).toBeVisible()
    await expect(page.getByTestId('username')).toBeVisible()
    await expect(page.getByTestId('login-submit')).toBeEnabled()
  }
  await page.getByTestId('username').fill(user)
  await page.getByTestId('password').fill(password)
  await page.getByTestId('login-submit').click()
  await expect(page.getByTestId('side-menu')).toBeVisible()
})
