CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id" bigint NOT NULL REFERENCES "users" ("id") ON DELETE CASCADE,
  "refresh_token" varchar NOT NULL UNIQUE,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "sessions" 
  ADD FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id") 
  ON DELETE CASCADE 
  DEFERRABLE INITIALLY IMMEDIATE;