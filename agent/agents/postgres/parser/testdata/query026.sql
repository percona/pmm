DELETE FROM tbl_scores
WHERE student_id IN
(SELECT student_id
FROM
(SELECT student_id,
ROW_NUMBER() OVER(PARTITION BY student_id
ORDER BY student_id) AS row_num
FROM tbl_scores_2) t
WHERE t.row_num <> 1);

