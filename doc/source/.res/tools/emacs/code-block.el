; -*- mode: lisp -*-
;; DOCUMENTATION:
;; Terminology:
;;
;;; - Code block signature
;;;   A code block signature is the command name followed by all of its
;;;   parameters with values removed.  The parameters must be separated
;;;   by a single whitespace character.  All non-alphabetical symbols
;;;   are replaced with a single dash (-) character. 
;;;
;;; - Code block ID
;;;   The code block ID applies the following rules to the supplied code block signature: 
;;;
;;;   1. White space characters are replaced with the dot (.) character.
;;;   2. The whole string is surrounded by the id limit marks.


;;; CONSTANTS

(defconst pmm-code-block ".. include:: .res/code/sh.org
   :start-after: %s
   :end-before: #+end-block")
(defconst pmm-id-limit-mark "+"
  "The symbol which appears to the left and to the right of the code block ID")
(defconst pmm-id-sep-mark "."
  "The symbol which separates tokens in the code block ID")
(defconst pmm-sig-sep-mark " "
  "The symbol which separates tokens in the code block signature")


;;; PRIVATE FUNCTIONS

(defun pmm-make-code-block-id (code-block-sig)
  "Transforms the code block signature into a code block ID."
  (concat pmm-id-limit-mark
	  (replace-regexp-in-string pmm-sig-sep-mark
				    pmm-id-sep-mark
				    code-block-sig)
	  pmm-id-limit-mark))

;;; INTERACTIVE FUNCTIONS

;; TODO: apply correct indentation
;; TODO: enable adding id not only signatures
;; TODO: automatically detect the level of nesting of the active file and change
;; the path to the .res/code/sh.org file.
;; TODO: search through all files under .res/code and insert the appropriate reference
(defun pmm-code-block (code-block-sig)
  "Inserts the code-block into the current document at the point of the cursor.
This function expects that a valid code block signature is supplied.
"
  (interactive "sCode block ID: ")
  (insert (format pmm-code-block (pmm-make-code-block-id code-block-sig))))

(defun pmm-insert-code-block-id (code-block-sig)
    "Inserts the code block ID at the point by transforming the
supplied code block signature."
    (interactive "sCode block signature: ")
    (insert (pmm-make-code-block-id code-block-sig)))
