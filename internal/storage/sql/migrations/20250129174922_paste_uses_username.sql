-- Modify "pastes" table
ALTER TABLE "public"."pastes" DROP CONSTRAINT "pastes_owner_fkey", ALTER COLUMN "owner" TYPE text, ADD CONSTRAINT "pastes_owner_fkey" FOREIGN KEY ("owner") REFERENCES "public"."users" ("username") ON UPDATE NO ACTION ON DELETE NO ACTION;
