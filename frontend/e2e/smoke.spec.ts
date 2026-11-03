import { expect, test, type Page } from '@playwright/test'

// Runs against the demo build by default, or a real stack when E2E_BASE_URL is set
// (then E2E_USER / E2E_PASSWORD must be an admin account).
const user = process.env.E2E_USER ?? 'admin'
const password = process.env.E2E_PASSWORD ?? 'admin123'
// The real stack starts with an empty database; tests that read demo seed data only run on the demo build.
const needsSeed = () => test.skip(!!process.env.E2E_BASE_URL, 'needs demo seed data')

async function login(page: Page, u = user, p = password) {
  await page.goto('/')
  await expect(page).toHaveURL(/\/login/)
  await page.getByTestId('username').fill(u)
  await page.getByTestId('password').fill(p)
  await page.getByTestId('login-submit').click()
}

test('both daily boards render KPI, table and empty state on an empty database', async ({ page }) => {
  await login(page)
  await expect(page).toHaveURL(/^https?:\/\/[^/]+\/$/)
  for (const [path, title] of [
    ['spray-daily', '喷涂日排产看板'],
    ['sheet-daily', '钣金日排程看板'],
  ]) {
    await page.goto(`/board/${path}`)
    await expect(page.getByRole('heading', { name: title })).toBeVisible()
    await expect(page.locator('.board .kpi')).toHaveCount(5)
    await expect(page.locator('.board table')).toBeVisible()
    await page.goto(`/board/${path}?fullscreen=1`)
    await expect(page.getByRole('heading', { name: title })).toBeVisible()
    await expect(page.getByTestId('side-menu')).toHaveCount(0)
  }
})

test('board seed plans appear in both boards', async ({ page }) => {
  needsSeed()
  await login(page)
  await expect(page).toHaveURL(/^https?:\/\/[^/]+\/$/)
  for (const path of ['spray-daily', 'sheet-daily']) {
    await page.goto(`/board/${path}`)
    await expect(page.locator('.board tbody tr').first()).toBeVisible()
  }
})

test('workshop daily and monthly reports render chart, total and comparison on an empty database', async ({ page }) => {
  await login(page)
  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('车间产量日报').click()
  await expect(page.getByTestId('workshop-total')).toContainText('合计')
  await expect(page.getByTestId('workshop-chart')).toBeVisible()
  await expect(page.getByTestId('workshop-table')).toBeVisible()
  await page.getByTestId('side-menu').getByText('车间产量月报').click()
  await expect(page.getByTestId('workshop-comparison')).toBeVisible()
  await expect(page.getByText('钣金车间环比')).toBeVisible()
})

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
  needsSeed()
  await login(page)
  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('扫描记录', { exact: true }).click()
  await expect(page).toHaveURL(/\/report\/scans/)
  const table = page.getByTestId('scan-table')
  await expect(table.locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  const barCode = (await table.locator('tbody tr.ant-table-row').first().locator('td').nth(4).innerText()).trim()
  await page.getByPlaceholder('条码 / 批次 / 工程 / 编号').fill(barCode)
  await page.getByRole('button', { name: /查\s*询/ }).click()
  const codeCells = table.locator('tbody tr.ant-table-row td:nth-child(5)')
  await expect(codeCells.filter({ hasNotText: barCode })).toHaveCount(0)
  await expect(codeCells).not.toHaveCount(0)
  await expect(codeCells).not.toHaveCount(20)
})

test('production balance page and progress summary toggle', async ({ page }) => {
  needsSeed()
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

test('quality pages list inspections and the scan page records a fail verdict', async ({ page }) => {
  needsSeed()
  await login(page)
  await page.getByTestId('side-menu').getByText('质量管理').click()
  await page.getByTestId('side-menu').getByText('质检扫描记录').click()
  await expect(page).toHaveURL(/\/quality\/qc-scans/)
  const table = page.getByTestId('qc-table')
  await expect(table.locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  await expect(page.getByTestId('qc-summary')).toContainText('待返工')

  await page.getByTestId('side-menu').getByText('质量报表').click()
  await expect(page).toHaveURL(/\/quality\/report/)
  await expect(page.getByTestId('quality-summary')).toContainText('合格率')
  await expect(page.getByTestId('quality-table').locator('tbody tr.ant-table-row')).not.toHaveCount(0)

  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('不良品日报').click()
  await expect(page).toHaveURL(/\/report\/defect-daily/)
  await expect(page.getByTestId('defect-table').locator('thead')).toContainText('钣金质检')

  // Scan page: picking a QC job reveals the verdict switch; a fail verdict needs no flow scan.
  await page.evaluate(() => localStorage.setItem('gz-scan-job', '25'))
  await page.goto('/production/scan')
  await expect(page.locator('.handheld .ant-select-selection-item')).toHaveText('25-钣金质检')
  const verdict = page.getByTestId('qc-result')
  await expect(verdict).toBeVisible()
  await verdict.getByText('不合格').click()
  await expect(page.getByTestId('defect-reason')).toBeVisible()
  await expect(page.getByRole('button', { name: '记录不合格' })).toBeVisible()
})

test('outsource orders and delivery plans pages', async ({ page }) => {
  needsSeed()
  await login(page)
  await page.getByTestId('side-menu').getByText('排程管理').click()
  await page.getByTestId('side-menu').getByText('外协单查看').click()
  await expect(page).toHaveURL(/\/scheduling\/outsource-orders/)
  await expect(page.locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  await expect(page.getByText('开料进度')).toBeVisible()
  await page.getByTestId('side-menu').getByText('发货管理').click()
  await page.getByTestId('side-menu').getByText('发货计划').click()
  await expect(page).toHaveURL(/\/delivery\/plans/)
  await expect(page.locator('tbody tr.ant-table-row')).not.toHaveCount(0)
  await page.getByRole('button', { name: '新建计划' }).click()
  await expect(page.getByText('选择未挂计划的发货单')).toBeVisible()
})

test('machine output summary and detail switch on an empty database', async ({ page }) => {
  await login(page)
  await page.getByTestId('side-menu').getByText('生产报表').click()
  await page.getByTestId('side-menu').getByText('机台产量').click()
  await expect(page).toHaveURL(/\/report\/machine-output/)
  await expect(page.getByTestId('machine-output-table')).toBeVisible()
  await page.getByTestId('machine-mode').getByText('明细').click()
  await expect(page.getByTestId('machine-output-table').locator('thead')).toContainText('批次')
})

test('create a machine in an own factory', async ({ page }) => {
  needsSeed()
  await login(page)
  await page.getByTestId('side-menu').getByText('系统管理').click()
  await page.getByTestId('side-menu').getByText('机台管理').click()
  await page.getByTestId('machine-new').click()
  const code = `E2E-M-${Date.now()}`
  await page.getByTestId('machine-code').fill(code)
  await page.getByTestId('machine-name').fill('端到端机台')
  await page.getByRole('button', { name: /确\s*定|OK/ }).click()
  await expect(page.getByTestId('machine-table')).toContainText(code)
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
