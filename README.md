Hermod
======
## About

Hermod is the messenger of the gods.  
It tracks deployments as they roll out and posts status updates into Slack.  
It does this by watching the kubernetes api for namespaces and deployments with the correct annotations. When a new deployment rollout begins and completes updates are posted to the Slack API.  

## To deploy hermod

See `example/` directory for sample kubernetes manifests.

## Add resources for hermod to track

1. Add a namespace for hermod to monitor:
```
apiVersion: v1
kind: Namespace
metadata:
  name: hermod-test-ns
  annotations:
    hermod.uswitch.com/slack: <slack-channel-to-receive-notifications>
```
2. Add a new deployment to track in that namespace:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    # commit SHA of this deployment, used in slack messages
    service.rvu.co.uk/vcs-ref: c48655970ce3485815638ff8e430d9c69588bed2 
    # link to git repo, used in slack messages
    service.rvu.co.uk/vcs-url: https://github.com/my-org/my-app
  labels:
    app: nginx
  name: nginx-deployment
  namespace: hermod-test-ns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: hermod
    spec:
      containers:
      - image: nginx
        name: nginx
```

## To build:
```
make
```  
Executable binaries will be created in the `bin/` directory.

## To run:
```
SLACK_TOKEN=XXXX make run
```

## Options

| flag  | default | description |
|---|---|---|
| --kubeconfig  | "" | Path to kubeconfig file (leave blank when running in k8s pod)  |
| --level | info | log level  |
| --repo-url-annotation  | hermod.uswitch.com/gitrepo | Annotation you will add to tracked deployments. This indicates the respository location and is used when publishing messages to slack. |
| --commit-sha-annotation  | hermod.uswitch.com/gitsha | Annotation you will add to tracked deployments. This indicates the commit SHA deployed and is used when publishing messages to slack. |
| --git-annotation-warning | false | option to enable warning level logs if previous annotations are missing |

## Metrics

Hermod exposes some prometheus metrics on port `2112` at `/metrics`.

| name  | description  | type |
|---|---|---|
| hermod_deployment_processed_total | The total number of deployments processed | Counter |
| hermod_deployment_success_total | The total number of successful deployments processed | Counter |
| hermod_deployment_failed_total | The total number of failed deployments processed | Counter |

## To make an update to hermod
1. Create a Pull request against this project's `master` branch.
2. When your changes are reviewed and merged we will tag and release a new version for you.
3. A new docker image will be automatically created with the given tag.
4. Update your kubernetes deployment to use the new image.

## Support

If you come across issues please raise an issue against the GitHub project. Include as much detail as you can (including relevant logs, hermod version, kubernetes version, etc) so we have the best chance of understanding what went wrong.

## Contributing

Contributions are always welcome. Please feel free to raise a pull request and we will review as soon as we can.
