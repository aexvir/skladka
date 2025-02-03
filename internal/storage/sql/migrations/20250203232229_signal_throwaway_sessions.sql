-- Modify "sessions" table
ALTER TABLE "public"."sessions" ADD COLUMN "throwaway" boolean NOT NULL DEFAULT false;
