ALTER TABLE IF EXISTS users
ADD COLUMN IF NOT EXISTS image_id int references image (id) default NULL;