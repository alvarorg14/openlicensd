ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_grace_period_days_check;
ALTER TABLE policies ADD CONSTRAINT policies_grace_period_days_check
    CHECK (grace_period_days >= 0);

ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_duration_days_check;
ALTER TABLE policies ADD CONSTRAINT policies_duration_days_check
    CHECK (duration_days IS NULL OR duration_days >= 1);

ALTER TABLE policies DROP CONSTRAINT IF EXISTS policies_max_activations_check;
ALTER TABLE policies ADD CONSTRAINT policies_max_activations_check
    CHECK (max_activations IS NULL OR max_activations >= 1);

ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_max_activations_check;
ALTER TABLE licenses ADD CONSTRAINT licenses_max_activations_check
    CHECK (max_activations IS NULL OR max_activations >= 1);

ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_validation_count_check;
ALTER TABLE licenses ADD CONSTRAINT licenses_validation_count_check
    CHECK (validation_count >= 0);

ALTER TABLE license_machines DROP CONSTRAINT IF EXISTS license_machines_validation_count_check;
ALTER TABLE license_machines ADD CONSTRAINT license_machines_validation_count_check
    CHECK (validation_count >= 0);
