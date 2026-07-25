export const Messages = {
  test: 'Test',
  testing: 'Testing…',
  testService: 'Test on service',
  testResultsTitle: 'Test results',
  testSuccess: 'Success',
  testFailure: 'Failed',
  testFindings: (count: number) =>
    count === 0 ? 'no findings' : `${count} finding${count === 1 ? '' : 's'}`,
  testFailed: 'Test failed',
  scriptOutput: 'Script output',
  closeResults: 'Close results',
};
