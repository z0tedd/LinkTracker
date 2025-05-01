-- liquibase formatted sql
-- changeset z0tedd:02-add-foreign-key-to-users-preferences

-- up:
ALTER TABLE users_preferences
    ADD CONSTRAINT fk_users_preferences_subscriptions
    FOREIGN KEY (sub_id)
    REFERENCES subscriptions(sub_id)
    ON DELETE CASCADE;

-- down:
--rollback ALTER TABLE users_preferences DROP CONSTRAINT IF EXISTS fk_users_preferences_subscriptions;
