-- v7 (compatible with v6+): Add lid column to device table
ALTER TABLE waSocket_device ADD COLUMN lid TEXT;
