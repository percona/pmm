;;; This module contains functions that enable replacing the table with AMI IDs.
;;; All interactive functions have the 'pmm-' prefix.

(defconst ami-regions '(("us-east-1"      . "US East (N. Virginia)")
			("us-east-2"      . "US East (Ohio)")
			("us-west-1"      . "US West (N. California)")
			("us-west-2"      . "US West (Oregon)")
			("ca-central-1"   . "Canada (Central)")
			("eu-west-1"      . "EU (Ireland)")
			("eu-central-1"   . "EU (Frankfurt)")
			("eu-west-2"      . "EU (London)")
			("eu-west-3"      . "EU (Paris)")
			("ap-southeast-1" . "Asia Pacific (Singapore)")
			("ap-southeast-2" . "Asia Pacific (Sydney)")
			("ap-northeast-2" . "Asia Pacific (Seoul)")
			("ap-northeast-1" . "Asia Pacific (Tokyo)")
			("ap-south-1"     . "Asia Pacific (Mumbai)")
			("sa-east-1"      . "South America (São Paulo)")
			("us-east-2"      . "US East (Ohio)")))
(defconst ami-table-row-offset 3)
(defconst ami-table-column-offset 5)
(defconst ami-table-padding-char 32)
(defconst new-row-pattern "* - %s\n")
(defconst new-column-important-pattern "- **%s**\n")
(defconst new-column-pattern "- %s\n")
(defconst ami-id-pattern "`%s <https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#Images:visibility=public-images;imageId=%s>`_")


(defun make-table-entry (ami-data)
  "AMI-DATA is a pair where the car is the location id such as us-east-1 and the cdr is the ami ID itself."
  (let* ((region-id (car ami-data))
	 (ami-id (cadr ami-data))
	 (ami-id-column (format ami-id-pattern ami-id ami-id))
	 (row-offset (make-string ami-table-row-offset ami-table-padding-char))
	 (column-offset (make-string ami-table-column-offset 
				     ami-table-padding-char)))
    (concat row-offset
	    (format new-row-pattern (cdr (assoc-string region-id ami-regions)))
	    column-offset 
	    (format new-column-important-pattern region-id)
	    column-offset 	
	    (format new-column-pattern ami-id-column))))


(defun parse-ami-data (range-start range-end)
  (let ((ami-data (buffer-substring-no-properties range-start 
						  range-end)))
    (mapconcat 'make-table-entry 
	       (mapcar (lambda (row) (split-string row " " t "\s-+"))
		       (split-string ami-data "\n" t "\s-+")) "\n")))


(defun pmm-insert-ami-table-data (range-start range-end)
  (interactive "r")
  (insert (parse-ami-data range-start 
			  range-end))
  (delete-region range-start
		 range-end))

