# Focused UI scaffold correction

The source-change rerun `npm test -- --reporter=dot src/__tests__/dashboard-permissions.spec.ts src/__tests__/layout-permissions.spec.ts` exited 1 with 6/7 tests passing. The sole failure was the existing tenant-switch test expecting `Pancake` conversation-channel counts under a `channels:r` tenant after the A02 contract assigns `conversations_by_channel` to `messages:r`. The implementation correctly hid that data; the stale assertion was corrected to expect no breakdown. The complete failed runner output is preserved in `ui-focused-tests-messages-projection-failure.log`.

The corrected focused suite then passed 7/7 in `ui-focused-tests-final.log`. This scaffold correction did not change product behavior or broaden the seed.
