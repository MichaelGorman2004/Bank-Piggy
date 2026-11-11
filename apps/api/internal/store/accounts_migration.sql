-- Lock each affected plan before deriving its replacement. Concurrent saves must
-- either finish first or see the incremented revision after this migration.
WITH legacy AS MATERIALIZED (
 SELECT id, revision, plan
 FROM households
 WHERE EXISTS (
  SELECT 1 FROM jsonb_array_elements(plan->'members') AS member
  WHERE NOT member ? 'accounts' OR member ? 'savingsCents' OR member ? 'investmentsCents'
 )
 FOR UPDATE
), migrated AS (
 SELECT id, revision, plan, (
  SELECT jsonb_agg(
   (member - 'savingsCents' - 'investmentsCents') ||
   CASE WHEN member ? 'accounts' THEN '{}'::jsonb
   ELSE jsonb_build_object('accounts', jsonb_build_array(
    jsonb_build_object('id', 'legacy-savings', 'name', 'Savings', 'type', 'savings',
     'balanceCents', member->'savingsCents'),
    jsonb_build_object('id', 'legacy-investments', 'name', 'Other investments', 'type', 'other_investment',
     'balanceCents', member->'investmentsCents')
   )) END
   ORDER BY position
  )
  FROM jsonb_array_elements(plan->'members') WITH ORDINALITY AS members(member, position)
 ) AS members
 FROM legacy
)
UPDATE households AS household
SET revision = migrated.revision + 1,
 plan = jsonb_set(jsonb_set(migrated.plan, '{members}', migrated.members),
  '{revision}', to_jsonb(migrated.revision + 1)),
 updated_at = now()
FROM migrated
WHERE household.id = migrated.id;
