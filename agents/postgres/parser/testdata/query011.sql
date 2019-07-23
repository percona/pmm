SELECT count(*) FROM (SELECT * FROM without_complaints
    INTERSECT
    SELECT * FROM credit_card_wo_complaints) ppg
