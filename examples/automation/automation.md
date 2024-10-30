# Automation Example

If you store the definition of all the charts you use as code, you might want to iterate and run this plugin for all of them, instead of running it manually for each chart.

This example demonstrates how to automate chart imports using a shell script.

## Prerequisites

Same as basic example, plus:
- [Charts list file](./charts.yaml)
- [Automation script](./import-charts.sh)

## Usage
Make the script executable:
```bash
chmod +x import-charts.sh
```
Run the automation script:
```bash
./import-charts.sh
```

This will process each chart in the configuration file, importing it and its images to your registry.

For advanced scenarios where you are handling many charts and/or registries, you may consider using [Helmper](https://github.com/ChristofferNissen/helmper) too.
