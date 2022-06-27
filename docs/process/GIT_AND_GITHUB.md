# Working with Git and GitHub

The golden rule

Commit and push to the feature branches daily. Use pull requests (in the draft state until they are ready for review) to show WIP. Saying "I'm coding feature X" on the daily call without pushed commits is not okay.

## Preparation

Make sure your `git user.name` is set, and `user.email` is set to your corporate (if you are Perconian) or personal email, i.e. john.doe@percona.com.

Use topic branches in the same repository instead of GitHub forks if you have such permissions.

Name topic branches like this: `PMM-1234-short-description`. Always start with `PMM-XXXX` or `SAAS-XXXX`. Never omit a short description. Separate words with dashes, not underscore. Use only lowercase letters.

If you are community contributor and there is no ticket exists, either create one or just use `short-description` style for your forked repo branch.

## Commit rules

Every commit message should use this commit template:
```
PMM-XXXX Short summary up to 50 characters.
 
Optional 72-character wrapped longer description.
```

There should be a blank line between short summary and longer description. Final dot in the title is optional – if you hate it, drop it :)

While addressing review comments push the requested changes as a new commit instead of amending the original commit and force pushing to Github.

## Pull Request rules

1. Before Creating or updating PR always run linters and tests locally.
2. Every PR should follow the template of a commit message.
3. If PR provides a prototype or a proof-of-concept, PR should be draft so it is not accidentally merged.
4. Start PR as draft ASAP to let everyone see the job you are doing.
5. Check-list of things to do prior to submitting a PR for review:
    - [x] All tests in the PR pass.
    - [x] Clean up your code removing unnecessary tokens, code which is commented out, etc.
    - [x] All required lint rules pass.
    - [x] New tests are written if it's possible for new changes.
    - [x] FB exists and is up-to-date.
    - [x] All tests in FB pass.
    - [x] Don't write unit tests that communicate with external services those tests belong in a separate API tests repo (This point might only be relevant to backend developers).
6. Rules for merging a PR:
    - [x] All required checks must pass.
    - [x] The reviewers (minimum 2) are set as makes sense for the code (DevOps, BE, FE, QA), and all need to approve the code after review.
    Should the person assigned as a reviewer not be able to review the PR in due time, they must un-assign themselves as a a reviewer of the said PR.
    - [x] If change requests were made, all changes need to be resolved and re-approved by the original reviewers.
    - [x] Task should have Ready for merge status
    - [x] If all conditions for a successful merge are met (checks pass, code reviewed and approved, changes resolved and re-approved), ONLY then the PR can me merged:
         * If a PR has Assignees, then the last approver should ping them in in the appropriate dev channel. Typically that means they want to merge several PRs in a feature build together.
         * If a PR has no Assignees, then the last approver must assign it to the author and ping them on Slack.
         * If required checks fail for a confusing reason or some non-required checks fail for a long time, ping the appropriate dev channel.
    - [x] A PR should be merged with the "Squash and Merge" button (the code owner should set this as the only option in the repository settings).
    - [x] Never leave PRs fully approved, unassigned, and not merged.
    - [x] Tasks with several PRs should be merged with minimum time interval between the merges.
7. Delete the branch after merging the PR (note: this step may be automated for some of our repos, i.e. a bot deletes the branch right after the merge).
8. If you need to re-request a review from someone, please use GitHub feature as displayed in Figure 1 below (use case: I made some changes after the PR was reviewed so I want to request another review from the same reviewer). Ping reviewer if it's not reviewed in 1 working day. 
9. If you open a Pull Request to fix a Bug - put an explanation in the Long Description about the Bug. It may contain information about how it affects the component. Also, it's good to have short references on how the problem is solved (now applicable only to mongodb_exporter and proxysql_exporter)
10. If you implement a new Feature or make an Improvement or Fix a Bug - add a link in the Long Description to the related PR for Feature Build.


[Check GitHub PR review documentation](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/requesting-a-pull-request-review).

Figure 1. Request another review from a reviewer.
![](https://docs.github.com/assets/cb-4714/images/help/pull_requests/request-re-review.png "")
