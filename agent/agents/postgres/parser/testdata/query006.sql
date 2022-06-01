SELECT ens.company, ens.state, ens.zip_code, ens.complaint_count
FROM (select company, state, zip_code, count(complaint_id) AS complaint_count
   FROM credit_card_complaints
   WHERE state IS NOT NULL
   GROUP BY company, state, zip_code) ens
INNER JOIN
(SELECT ppx.company, max(ppx.complaint_count) AS complaint_count
 FROM (SELECT ppt.company, ppt.state, max(ppt.complaint_count) AS complaint_count
       FROM (SELECT company, state, zip_code, count(complaint_id) AS complaint_count
             FROM credit_card_complaints_2
             WHERE company = 'Citibank'
              AND state IS NOT NULL
             GROUP BY company, state, zip_code
             ORDER BY 4 DESC) ppt
       GROUP BY ppt.company, ppt.state
       ORDER BY 3 DESC) ppx
 GROUP BY ppx.company) apx
ON apx.company = ens.company
AND apx.complaint_count = ens.complaint_count
ORDER BY 4 DESC;
