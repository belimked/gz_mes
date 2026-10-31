-- +goose Up
UPDATE menu SET icon = 'home' WHERE path = '/system/factories';
UPDATE menu SET icon = 'team' WHERE path = '/system/users';
UPDATE menu SET icon = 'project' WHERE path = '/sale/projects';
UPDATE menu SET icon = 'contacts' WHERE path = '/sale/customers';
UPDATE menu SET icon = 'file-text' WHERE path = '/sale/contracts';
UPDATE menu SET icon = 'calendar' WHERE path = '/scheduling/batches';
UPDATE menu SET icon = 'barcode' WHERE path = '/production/labels';
UPDATE menu SET icon = 'swap' WHERE path = '/production/handovers';
UPDATE menu SET icon = 'scan' WHERE path = '/production/scan';
UPDATE menu SET icon = 'truck' WHERE path = '/delivery/notes';
UPDATE menu SET icon = 'calculator' WHERE path = '/finance/batch-valuation';
UPDATE menu SET icon = 'audit' WHERE path = '/finance/deliver-accounts';

-- +goose Down
UPDATE menu SET icon = '' WHERE parent_id IS NOT NULL;
