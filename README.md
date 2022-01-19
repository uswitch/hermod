Hermod
======
## About

[Hermod](https://en.wikipedia.org/wiki/Herm%C3%B3%C3%B0r) takes its name from the Norse messenger of the gods.  
It tracks deployments as they roll out and posts useful status updates into Slack.  
It does this by watching the kubernetes api for namespaces and deployments with the correct annotations. When a new deployment rollout begins and completes updates are posted to the Slack API.  
Any errors during the deployment rollout are captured and included in the Slack message (see example below). This can be very useful to help quickly debug a failing deployment.

Hermod will notify when a new deployment starts rolling out:  
![Deployment started notification](images/deploy-start.png?raw=true "Deployment started")  

When a deployment successfully rolls out:  
![Deployment succeeded notification](images/deploy-end-success.png?raw=true "Deployment succeeded")  

When a deployment fails to roll out:  
![Deployment failed notification](images/deploy-end-fail.png?raw=true "Deployment failed")  

## To deploy Hermod

1. Deploy Hermod to your kubernetes cluster  
    See `example/` directory for sample kubernetes manifests.  
2. Install Hermod app in your Slack directory [here](https://slack.com/apps).
3. Add Hermod to any channels in which you want to receive updates from Hermod.
4. Create resources for hermod to track.

## Resource annotations

### Namespace annotations

| annotation | example | description | notes |
|---|---|---|---|
| `hermod.uswitch.com/slack` | "hermod-updates" | Configures which Slack channel to post updates to | Required for each namespace that hermod should monitor. Different namespaces can send updates to different Slack channels |
| `hermod.uswitch.com/alert` | "failure" | Only notify on deployment rollout failure | Optional |

### Deployment annotations

| annotation | example | description |
|---|---|---|
| `hermod.uswitch.com/gitsha` | 2dafeb708437f6e537d19556d461e30aa96d4244 | Optional. Commit SHA of code deployment. The name of this annotation is configurable, see [here](#options). |
| `hermod.uswitch.com/gitrepo` | https://github.com/my-org/my-app | Optional. Git Repo Url of code deployment. The name of this annotation is configurable, see [here](#options). |

## Add resources for Hermod to track

1. Annotate or create a namespace for hermod to monitor:
```
apiVersion: v1
kind: Namespace
metadata:
  name: hermod-test-ns
  annotations:
    # The presence of this annotation enables hermod for the namespace and configures where to post updates in Slack
    hermod.uswitch.com/slack: <slack-channel-to-receive-notifications> 
    # Optional, notify only on deployment failure
    hermod.uswitch.com/alert: failure 
```
2. Add a new deployment in that namespace for hermod to monitor:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    # commit SHA of this deployment, used in slack messages, optional
    hermod.uswitch.com/gitsha: c48655970ce3485815638ff8e430d9c69588bed2 
    # link to git repo, used in slack messages, optional
    hermod.uswitch.com/gitrepo: https://github.com/my-org/my-app
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
        app: nginx
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

## Environment Variables

| name  | default | required | description |
|---|---|---|---|
| SLACK_TOKEN | "" | y | API token for Slack |
| SENTRY_ENDPOINT | "" | n | [Sentry DSN](https://docs.sentry.io/product/sentry-basics/dsn-explainer/) |
| CLUSTER_NAME | "" | n | Name of your kubernetes cluster, used in Slack messages |

## Monitoring

### Prometheus
Hermod exposes some prometheus metrics on port `2112` at `/metrics`.

| name  | description  | type |
|---|---|---|
| hermod_deployment_processed_total | The total number of deployments processed | Counter |
| hermod_deployment_success_total | The total number of successful deployments processed | Counter |
| hermod_deployment_failed_total | The total number of failed deployments processed | Counter |

### Sentry
Hermod can also publish some error events to [Sentry](https://sentry.io).  
This is optional but can be enabled by configuring the `SENTRY_ENDPOINT` environment variable and setting it to your [Sentry DSN](https://docs.sentry.io/product/sentry-basics/dsn-explainer/).  

## To make an update to Hermod
1. Create a Pull request against this project's `main` branch.
2. When your changes are reviewed and merged we will tag and release a new version for you.
3. A new docker image will be automatically created with the given tag.
4. Update your kubernetes deployment to use the new image.

## Support

If you come across issues please raise an issue against the GitHub project. Include as much detail as you can (including relevant logs, hermod version, kubernetes version, etc) so we have the best chance of understanding what went wrong.

## Contributing

Contributions are always welcome. Please feel free to raise a pull request and we will review as soon as we can.

## More like this
Check our other open source projects here:
https://www.rvu.co.uk/open-source
