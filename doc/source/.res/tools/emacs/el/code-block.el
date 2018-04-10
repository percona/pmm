;; -*- mode: lisp -*-
;; DOCUMENTATION:
;; Terminology:
;;
;; - Code block signature
;;   A code block signature is the command name followed by all of its
;;   parameters with values removed.  The parameters must be separated
;;   by a single whitespace character.  All non-alphabetical symbols
;;   are replaced with a single dash (-) character. 
;;
;; - Code block ID
;;   The code block ID applies the following rules to the supplied code block signature: 
;;
;;   1. White space characters are replaced with the dot (.) character.
;;   2. The whole string is surrounded by the id limit marks.

;; proc replace a code block from document to resources
;;   if  the cursor is on an indented line
;; then search up for the line that contains ".. code-block"
;;   if the cursor is on the line that contains ".. code-block"
;; then mark this line
;; else
;;   if the cursor is not on an indented line
;; then search down
;; else detect the code block type
;;      search for the first unindented line
;;      mark the end of the selection
;;      request signature
;;      generate id
;;      save code block to the named register
;;      replace the code block with the include directive
;;      save the cursor position
;;      visit the resource file for the code block type
;;      search for the signature
;;   if the full signature is found
;; then move the cursor to the signature
;; exit 
;; else search for the most specific heading
;;   if the most specific heading is found
;; then move the cursor to the heading
;; exit
;; else move the cursor to the beginning of file
;; exit

(defun pmm-code-block-replace ()
  "Replaces the selected code block to its resource file and inserts the appropriate include directive"
  ;; todo find the next code block
  ;; todo detect type
  ;; todo save code block to buffer
  ;; todo request signature and id
  ;; todo ins inlude directive

  )

;; proc insert a code block from the named register
;;   do read the signature
;;  and read the id
;;  and insert org block
;;  and insert the contents of the named register
;;  and open the document
(defun pmm-code-block-insert-register ()
  "Inserts an code block from the predefined register on the current line")

;;; CONSTANTS
(defvar pmm-doc-source-dir "")
(defconst resource-catalog ".res/"
  "The name of the directory which contains resources. These are
  elements other than text, headings, or lists) which are included
  into documents" )
(defconst code-block-catalog (concat resource-catalog "code/"))
(defconst code-block-format ".org" "the extension of files which contain code blocks")

(defun code-block-catalog (code-block-type)
  "Creates a relave path to the file which contains a
collection of code block elements. All these elements belong to
CODE-BLOCK-TYPE. Such files must be in the CODE-BLOCK-FORMAT."
  
  (concat code-block-catalog code-block-type code-block-format))

(defconst sql-code-blocks   (code-block-catalog "sql"))
(defconst sh-code-blocks    (code-block-catalog "sh"))
(defconst yaml-code-blocks  (code-block-catalog "yaml"))
(defconst js-code-blocks    (code-block-catalog "js"))

(defconst pmm-code-block-pattern ".. include:: .res/code/%s.org
   :start-after: %s
   :end-before: #+end-block")

(defconst pmm-id-limit-mark "+"
  "The symbol which appears to the left and to the right of the code block ID")
(defconst pmm-id-sep-mark "."
  "The symbol which separates tokens in the code block ID")
(defconst pmm-sig-sep-mark " "
  "The symbol which separates tokens in the code block signature")

;; PRIVATE FUNCTIONS

(defun pmm-make-code-block-id (code-block-sig)
  "Transforms the code block signature into a code block ID."
  (concat pmm-id-limit-mark
	  (replace-regexp-in-string pmm-sig-sep-mark
				    pmm-id-sep-mark
				    code-block-sig)
	  pmm-id-limit-mark))

;; INTERACTIVE FUNCTIONS

;; TODO: apply correct indentation
;; TODO: enable adding id not only signatures
;; TODO: automatically detect the level of nesting of the active file and change
;; the path to the .res/code/sh.org file.
;; TODO: search through all files under .res/code and insert the appropriate reference

(defun pmm-code-block-pattern (code-block-sig)
  "Inserts the code-block into the current document at the point of the cursor.
This function expects an existing code block signature or signature.
"
  (interactive "sCode block ID: ")
  (insert (format pmm-code-block-pattern (pmm-make-code-block-id code-block-sig))))

(defun pmm-insert-code-block-id (code-block-sig)
    "Inserts the code block ID at the point by transforming the supplied code block signature."
    (interactive "sCode block signature: ")
    (insert (pmm-make-code-block-id code-block-sig)))

(defconst short-token-length 7)
(defun make-id-signature (keyword-token &optional longp)
  "Takes the identifier of a text object and produces its unique
signature truncated to the length specified in the SHORT-TOKEN-LENGTH
constant. If LONGP is true than the produced token is not truncated."

  (let ((signature (sha1 keyword-token)))
    (if longp signature
      (substring signature 0 short-token-length))))






