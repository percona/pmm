;;; This module contains functions that enable inserting and verifying of headings
;;; All interactive functions have the 'pmm-' prefix.

(defconst directive-name-ref "ref")
(defconst directive-delimiter ":")
(defconst directive-contents-delimiter "`")
(defconst directive-target-delimiter-start "<")
(defconst directive-target-delimiter-end ">")

(defun make-directive (directive-name directive-contents &optional directive-target)
  (concat directive-delimiter
	  directive-name
	  directive-delimiter
	  directive
	  
    )

(defun pmm-apply-ref ()
  "Wraps the current heading into a ref directive. The effective heading becomes the desplayed contents and the heading ID becomes the link target. Then it corrects the underlining to match the new heading.

This function assumes the following:

1. The cursor is somewhere in the heading text
2. This heading has an ID

"
  (interactive)
  
  )
