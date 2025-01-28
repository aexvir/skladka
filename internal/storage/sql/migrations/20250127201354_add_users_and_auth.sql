-- Create "sessions" table
CREATE TABLE "public"."sessions" ("id" bigserial NOT NULL, "token" uuid NOT NULL, "username" text NOT NULL, "data" jsonb NOT NULL, "created_at" timestamp NOT NULL DEFAULT now(), "expires_at" timestamp NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "sessions_token_key" UNIQUE ("token"));
-- Create "users" table
CREATE TABLE "public"."users" ("id" bigserial NOT NULL, "username" character varying(255) NOT NULL, "uuid" uuid NOT NULL, "credentials" bytea NOT NULL, "created_at" timestamp NOT NULL DEFAULT now(), "updated_at" timestamp NULL, "deleted_at" timestamp NULL, PRIMARY KEY ("id"), CONSTRAINT "users_username_key" UNIQUE ("username"), CONSTRAINT "users_uuid_key" UNIQUE ("uuid"));
-- Modify "pastes" table
ALTER TABLE "public"."pastes" ADD COLUMN "owner" bigint NULL, ADD CONSTRAINT "pastes_owner_fkey" FOREIGN KEY ("owner") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
