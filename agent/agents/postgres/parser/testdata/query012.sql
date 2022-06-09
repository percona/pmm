SELECT count(*) FROM (SELECT * FROM without_complaints
    EXCEPT
SELECT * FROM credit_card_wo_complaints) ppg
