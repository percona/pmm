export const Messages = {
  columns: {
    serviceId: 'Service ID',
    serviceName: 'Service Name',
    nodeName: 'Node Name',
    port: 'Port',
    cluster: 'Cluster',
  },
  actions: {
    add: 'Add Service',
    edit: 'Edit',
    dashboard: 'Dashboard',
    qan: 'QAN',
  },
  details: {
    serviceId: 'Service ID',
  },
  addressInfo:
    'The address pmm-agent uses to connect to this database, as seen from the host where that pmm-agent runs. 127.0.0.1 or localhost means the database runs on that same host, not necessarily on PMM Server.',
  learnMore: 'Learn more',
  empty: 'No services available',
  nonClustered: 'Non-clustered services',
  organizeByClusters: 'Organize by Clusters',
  delete: {
    title: 'Confirmation required',
    warning: 'Attention! This action is irreversible!',
    forceDescription: 'Force mode is going to delete all associated agents.',
    confirm: (count: number) => `Yes, delete service${count > 1 ? 's' : ''}`,
    message: (serviceNames: string[]) =>
      serviceNames.length === 1
        ? `By deleting service “${serviceNames[0]}” you will not be able to recover it. Are you sure you want to delete it?`
        : `Are you sure that you want to permanently delete ${serviceNames.length} services?`,
    success: (deleted: number, serviceNames: string[]) =>
      serviceNames.length === 1
        ? `Service “${serviceNames[0]}” was removed effectively`
        : `${deleted} of ${serviceNames.length} services successfully deleted`,
  },
};
