
;; TODO: Implement inserting given text above or strictly below a section
;; TODO: Insert headings via a command:
;;; 1. From the supplied string determine how to use replace directives
;; TODO: Automatically create section identifiers for the sections that describe metic
;; TODO: Create a template for a metric description
;; TODO: Allow one or more whitespace in replace patterns
;; Constants

(defconst pmm-replace-dir/pattern ".. |%s| replace::")
(defconst pmm-replace-dir/pattern/ref ".. |%s| replace:: :ref:`%s`"
  "A pattern for replace directive which inserts a reference to a section")
(defconst pmm-heading/decorator-pattern "^[%s]+$"
  "A regular expression which represents the RST syntax for heading 2")
(defconst pmm-h1-symbol "=")
(defconst pmm-h2-symbol "-")
(defconst pmm-ref/this-dashboard "this-dashboard"
  "The name of a replace directive which is used to represent the open dashboard")

; Utility functions

(defun pmm-point/heading (heading-level above-p)
  "Move the pointer to the location of the heading. 

If ABOVE-P is not nil then the pointer is moved to the next heading in text."

  (let (level)
    (cond ((eq heading-level 1) (setq level pmm-h1-symbol))
	  ((eq heading-level 2) (setq level pmm-h2-symbol)))
    (if (null above-p)
	(search-backward-regexp (format pmm-heading/decorator-pattern level))
      (search-forward-regexp    (format pmm-heading/decorator-pattern level)))))

(defun pmm-make/replace-dir/ref/dashboard (dashboard-name)
  "Computes the ID of the given dashboard"
  (format pmm-replace-dir/pattern/ref
	  dashboard-name
	  (file-name-sans-extension
	   (file-name-nondirectory
	    (buffer-file-name)))))

(defun pmm-make/normalize ()
  "Normalizes the heading to make it useful as an ID:

1. Brings the whole text to lower case
2. Replaces all non-text symbols with whitespace
3. Replaces all consecutive whitespace characters with a single whitespace character."

 
(defun pmm-metric-id ()
  "Computes the metric ID based on the context of the given dashboard.
This function tries to detect a heading above assuming it is the name of a metric. 

Then, it returns the full metric ID which can be inserted anywhere in text. 

The caller must place the cursor in the appropriate location in the file and then insert the ID."

  (let ((dashboard-id pmm-make/replace-dir/ref/dashboard pmm-ref/this-dashboard))
    (pmm-point/heading 2 t)
    (line-move -1)

  

; Interactive functions

(defun pmm-insert/replace-dir/ref/this-dashboard ()
  "Inserts a replacement directive named this-dashboard to refer to the current dashboard."
  (interactive)
  (search
  
  (insert (pmm-make/replace-dir/ref/dashboard pmm-ref/this-dashboard)))

(defun pmm-template-metric ()
  "Inserts a section which represents a metric"
  (interactive))

