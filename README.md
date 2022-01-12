Hermod
======
## About

Hermod is the messenger of the gods.  
It tracks deployments as they roll out and posts status updates into Slack.  
It does this by watching the kubernetes api for namespaces and deployments with the correct annotations. When a new deployment rollout begins and completes updates are posted to the Slack API.  

## To deploy hermod

** Example hermod deployment yaml file **

** Example annotated namespace yaml file **
** Example annotated deployment yaml file **

## To build the project:
run: `make`

## To make an update to hermod
1. Create a Pull request against this project's `master` branch.
2. When your changes are reviewed and merged we will tag and release a new version for you.
3. A new docker image will be automatically created with the given tag.
4. Update your kubernetes deployment to use the new image.

## Support

If you come across issues please raise an issue against the GitHub project. Include as much detail as you can (including relevant logs, hermod version, kubernetes version, etc) so we have the best chance of understanding what went wrong.

## Contributing

Contributions are always welcome. Please feel free to raise a pull request and we will review as soon as we can.
