-- Dropping the table forgets every mail still waiting for a send: those users ask for another.
DROP INDEX IF EXISTS mail_outbox_due_at_idx;

DROP TABLE IF EXISTS mail_outbox;
