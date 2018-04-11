(defconst min-page-width 80)
(defconst shortest-underlining-length 3)
(defconst heading-pattern "^[%s]\\{%s\\}")

(defconst heading-level-section ?=)
(defconst heading-level-subsection ?-)
(defconst heading-underline-characters [(char-to-string heading-level-section)
					(char-to-string heading-level-subsection)])
(defconst heading-id-pattern "^[.]\\{2}\\[_]\s-[^[:blank:]]+[^:][:]$"

(defun heading-rule (underlinep overlinep prefix line-char))

(defun heading-underline-characters ()
  "Concatenate all heading underline characters."

  (mapconcat 'identity heading-underline-characters ""))

(defun next-heading-underlining ()
  "Sets the underlining to be at least the default page width"

  (search-forward-regexp (format heading-pattern
				 (heading-underline-characters)
				 shortest-underlining-length)
			 nil t))
  
(defun normalize-heading-text (underlining-length next-check)
  "Verifies if the text on the current line is a valid heading."

  (move-beginning-of-line 1)

  (let* ((starts-with-blank-p   (search-forward-regexp "^[[:blank:]]" nil t))
	 (ends-with-blank-p     (search-forward-regexp "[[:blank:]]$" nil t))
	 (start-of-heading-text (point))
	 (end-of-heading-text   (progn (move-end-of-line 1) (point)))
	 (heading-length        (- end-of-heading-text 
				   start-of-heading-text))
	 (text-too-long-p       (> heading-length 
				   underlining-length)))
    
    (cond (starts-with-blank-p (progn (move-beginning-of-line 1) 
				      (delete-char 1)
				      (normalize-heading-text underlining-length 
							      next-check)))
	  (ends-with-blank-p (progn (move-end-of-line 1)
				    (delete-char -1)
				    (normalize-heading-text underlining-length
							    next-check)))
	  (text-too-long-p (normalize-heading-text (normalize-underlining heading-length) next-check))
	  (t (funcall next-check)))))

(defun normalize-underlining (heading-length)
  "Improves the heading underlining."

  (move-beginning-of-line 1)

  (let* ((current-line (lambda () (count-lines (point-min) (point))))
	 (init-line-number  (funcall current-line)))
      (next-heading-underlining)
      (if (> (- (funcall current-line) 
		init-line-number)
	     1)
	  (funcall current-line)
	13)))

(defun check-overlining ()
  "Stub"
  (identity 42))

;; Inserting a new heading
(defun make-heading-at-point (heading-text heading-id heading-level)
  "Combines heading components"
  (concat "\n" heading-id
	  "\n\n" heading-text
	  "\n" (make-string min-page-width heading-level)
	  "\n\n"))

(defun pmm-insert-heading (heading-level)
  "Inserts a heading"
  (interactive "nHeading level: (1 or 2): ")
  (beginning-of-line)
  (let (level)
    (cond ((= heading-level 1) (setq level heading-level-section))
	  ((= heading-level 2) (setq level heading-level-subsection))
	  (t (setq level heading-level-section)))
    (insert (make-heading-at-point (call-interactively 'make-heading-text) 
				   (call-interactively 'make-heading-id) level))))

(defun make-heading-text (heading-text)
  "Enter the heading text truncating to the page width if necessary"
    (interactive "sHeading text: ")
    (if (> (length heading-text) min-page-width)
	(substring heading-text 0 min-page-width)
      heading-text))

;;      todo » Process the current line
;;        if » the current line is not empty
;;      then » use the current line as the text of heading
;;       and » request the ID

(defconst heading-id-prefix ".. _")
(defconst heading-id-suffix ":")
(defun make-heading-id (heading-id)
  "Create a heading id"
  (interactive "sHeading ID: ")
  
  (downcase (concat heading-id-prefix
		    heading-id
		    heading-id-suffix)))

