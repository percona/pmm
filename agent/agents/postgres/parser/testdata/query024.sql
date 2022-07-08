UPDATE reward_members
SET member_status = (
CASE member_status
WHEN 'gold' THEN 'gold_group'
WHEN 'bronze' THEN 'bronze_group'
WHEN 'platinum' THEN 'platinum_group'
WHEN 'silver' THEN 'silver_group'
END
)
WHERE member_status IN ('gold', 'bronze','platinum', 'silver');
