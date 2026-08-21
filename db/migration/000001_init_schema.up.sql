CREATE TYPE "user_status" AS ENUM (
  'active',
  'suspended',
  'deleted'
);

CREATE TYPE "auth_provider" AS ENUM (
  'google',
  'apple',
  'local_email'
);

CREATE TYPE "gender" AS ENUM (
  'male',
  'female',
  'other'
);

CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "display_name" varchar,
  "avatar_url" varchar,
  "birthdate" date,
  "gender" gender,
  "status" user_status NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  "deleted_at" timestamptz
);

CREATE TABLE "user_identities" (
  "id" bigserial PRIMARY KEY,
  -- Changed from bigserial to bigint to correct the FK definition
  "user_id" bigint NOT NULL, 
  "provider" auth_provider NOT NULL,
  "provider_sub" varchar NOT NULL,
  "provider_email" varchar,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "password_credentials" (
  "id" bigserial PRIMARY KEY,
  "email" varchar NOT NULL UNIQUE,
  "email_verified_at" timestamptz,
  "user_id" bigint NOT NULL, 
  "password_hash" varchar NOT NULL,
  "password_algo" varchar NOT NULL DEFAULT 'argon2id',
  "password_updated_at" timestamptz NOT NULL DEFAULT (now()),
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

-- Indexing email is still good for prefix searches, but the UNIQUE constraint above already creates an index.
-- If you want case-insensitive email unique constraints, you can use:
-- CREATE UNIQUE INDEX users_email_lower_idx ON users (LOWER(email));

CREATE UNIQUE INDEX ON "user_identities" ("provider", "provider_sub");

CREATE UNIQUE INDEX ON "password_credentials" ("user_id");

-- Added ON DELETE CASCADE so deleting a user automatically cleans up credentials and identities
ALTER TABLE "user_identities" 
  ADD FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id") 
  ON DELETE CASCADE 
  DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "password_credentials" 
  ADD FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id") 
  ON DELETE CASCADE 
  DEFERRABLE INITIALLY IMMEDIATE;