# Data Open Studio

Data Open studio is designed as a suite of tools to help you with your data pipelines.

The project is in early stages of development and is divided in 2 main parts:

- **Pipeline**: A tool to help you build your data pipelines. It will be a visual tool that will allow you to drag and drop components to build your pipeline.
- **Viz**: A tool to help you visualize your data. It will be a visual tool that will allow you to create dashboards and reports.

## Roadmap

To date the basic features of pipeline have been implemented.
- **Pipeline**: 
  - Drag and drop components
  - Save and load pipelines
  - Execute pipelines
  - Schedule pipelines
- **Viz**: 
  - Storage engine
  - Language of query

The next steps are:
- **Pipeline**:
    - Add more components like SFTP node, S3 node, and outputs node sql or csv
    - Refactor emails into a separate component so any pipeline can be used as an email sender
    - Add support for monitoring pipelines
    - Polish map nodes
- **Viz**:
    - Add LSP for resin
    - Add support for multiple inputs
    - Test storage engine
    - Test language of query
    - Add support for visualizations
    - Add support for dashboards
    - Add support for reports
- **General**:
    - Add E2E tests for both pipeline and viz
    - Add more documentation
    - Add more examples once the pipeline is stable
    - Add more tests
    - Refactor gatewway to be able to control identity, auth and session management for both pipeline and viz
    - Add more logging and monitoring to both pipeline and viz with grafana and prometheus
    - Add support for multiple users and teams
    - Add support for role-based access control