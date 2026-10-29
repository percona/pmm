// A dashboard as Grafana's search lists it; `url` carries Grafana's sub path,
// such as /graph/d/<uid>/<slug>.
export interface StarredDashboard {
  uid: string;
  title: string;
  url: string;
}
