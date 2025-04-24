# About label based access control (LBAC) in PMM

Access control in PMM allows you to manage access to data. By using access control you can restrict access to monitoring metrics and Query Analytics data. This is particularly important in environments where sensitive data is involved, and it helps ensure that only authorized users can access specific information, which is crucial for maintaining security and compliance.

Access control is implemented through the use of roles assigned to each individual user. Every role can have a set of labels, or key-value pairs, that are used to filter and restrict access to specific data. For example, you can create a selector for a specific environment ("environment=prod") or database type ("service_type=mysql") and assign the selector to a certain role. Then, you can assign that role to one or more users, allowing those users to access only the relevant data.

Therefore, access control provides a standardized way of granting, changing, and revoking access to data based on a set of roles assigned to the user.

The following topics are covered as part of access control:

- [Enable access control](enable_access_control.md)
- [Labels for access control](labels.md)
- [Create access roles](create_roles.md)
- [Use cases](use_cases.md)