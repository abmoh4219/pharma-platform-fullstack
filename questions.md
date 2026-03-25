# Business Logic Questions Log

1. RBAC roles and data scopes
   ○ Question: Prompt lists 4 roles and "data scopes by institution/department/team" but doesn't give exact permissions or how scopes are assigned.
   ○ My Understanding: System admin sees everything; other roles see only their scope.
   ○ Solution: Added role + scope fields to users table and checked in every API.

2. Candidate match score 0-100
   ○ Question: How exactly to calculate the score and show explainable reasons?
   ○ My Understanding: Weighted score (skills 40%, experience 30%, etc.).
   ○ Solution: Built a scoring function in backend that returns score + reasons list.

3. Duplicate resume merge on bulk import
   ○ Question: Same phone or ID number → how to merge?
   ○ My Understanding: Merge fields into one record, keep history.
   ○ Solution: Check on import, update existing record.

4. Case unique number and 5-minute duplicate block
   ○ Question: Exact format and how to block duplicates?
   ○ My Understanding: YYYYMMDD-institution-6digit serial + timestamp check.
   ○ Solution: Generated in backend + 5-min lock.

5. Expiration reminder + auto-deactivate
   ○ Question: How to highlight and deactivate?
   ○ My Understanding: Background job or on-load check (red if <30 days).
   ○ Solution: Added cron-like check in backend + UI highlight.