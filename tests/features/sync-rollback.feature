Feature: stas sync with rollback trigger

    @nomock @todo-fix-in-mock
    Scenario: sync with rollback trigger
        Given file "cfg.yaml" exists:
            """
            stacks:
                stack1:
                    name: stastest-%scenarioid%
                    path: tpls/stack1.yml
                    tags:
                        STAS_TEST: '%featureid%'

            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              RollbackAlarm:
                Type: AWS::CloudWatch::Alarm
                Properties:
                  AlarmName: !Ref AWS::StackName
                  Namespace: !Sub "${AWS::StackName}-whatever"
                  MetricName: Errors
                  Statistic: Maximum
                  Period: '60'
                  EvaluationPeriods: '1'
                  Threshold: 0
                  ComparisonOperator: GreaterThanThreshold
                  ActionsEnabled: yes
                  # the next config keeps alarm in `ALARM` state always
                  TreatMissingData: breaching
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "cfg.yaml":
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
                  sometag: foo
                rollbackConfiguration:
                  monitoringTimeInMinutes: 1
                  rollbackTriggers:
                    - arn: arn:aws:cloudwatch:{{ .AWS.Region }}:{{ .AWS.AccountID }}:alarm:stastest-%scenarioid%
                      type: AWS::CloudWatch::Alarm

            """
        And I run "sync -c cfg.yaml --no-interaction"
        Then exit code should not be zero
        And output should contain:
        """
        The following CloudWatch Alarms were in ALARM state
        """
        And stack "stastest-%scenarioid%" should have status "UPDATE_ROLLBACK_COMPLETE"
