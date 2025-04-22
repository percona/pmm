# About label based access control (LBAC) in PMM

Access control in PMM allows you to manage access to data. By implementing Access control you can restrict access to certain data or features. This is particularly important in environments where sensitive data is involved, and it helps ensure that only authorized users can access specific information.

Access control is implemented through the use of roles assigned to each individual user. Every role can have a set of labels, or key-value pairs, which act as metrics selectors. These labels can be used to filter and restrict access to specific metrics. For example, you can create a selector for a specific environment ("environment=prod") or database type ("service_type=mysql") and assign the selector to a certian role. Then, you can assign that role to one or more users, allowing only users with those roles to see the corresponding metrics.

This approach gives a fine-grained control over who can access what data, and ensure that users only see the information they need to. It also helps prevent unauthorized access to sensitive data, which is crucial for maintaining security and compliance.

In summary, access control provides a standardized way of granting, changing, and revoking access to data based on a set of roles assigned to the users.

The following topics are covered as part of access control:

- [Enable access control](enable_access_control.md)
- [Labels for access control](labels.md)
- [Create access roles](create_roles.md)
- [Use cases](use_cases.md)