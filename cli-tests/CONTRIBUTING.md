# Contributing

### Project Licenses

- All modules use [Apache License v2.0](LICENSE.md).

## Test suite structure

### Assertions

* **Assertion can be used directly in a test.**  
  Playwright provides wide [matchers list](https://playwright.dev/docs/test-assertions) that can be used directly inside a test.
  


* **Declaration of a test variable** should be done on top of the test

### Test Grouping

* **One feature per spec file.**  
This will help to split tests and order the execution for workers parallel mode. It will provide control over test execution time and increase maintainability of test suite.


## Coding Conventions

### Naming Conventions

* **Acronyms**  
  Whenever an acronym is included as part of a type name or method name, keep the first
  letter of the acronym uppercase and use lowercase for the rest of the acronym. Otherwise,
  it becomes potentially very difficult to read or reason about the element without
  reading documentation (if documentation even exists).

  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Consider for example a use case needing to support an HTTP URL. Calling the method
  `getHTTPURL()` is absolutely horrible in terms of usability; whereas, `getHttpUrl()` is
  great in terms of usability. The same applies for types `HTTPURLProvider` vs
  `HttpUrlProvider`.  
  |  
  Whenever an acronym is included as part of a field name or parameter name:
  * If the acronym comes at the start of the field or parameter name, use lowercase for the entire acronym, ex: `url; id;`
  * Otherwise, keep the first letter of the acronym uppercase and use lowercase for the rest of the acronym, ex: `baseUrl; userId;`


* **Methods.**   
  * Methods should be named as actions with camelCase (changeSorting, changeGrouping, etc..)
  * General preferable declaration style is _fat arrow function_,  
    ex: `const add = (a, b) => a + b;`


* **Test Files.**   
  Test files should be named with camelCase and auxiliary or search part should be added with dash, ex: `mongoDb-integration.spec.ts;`
