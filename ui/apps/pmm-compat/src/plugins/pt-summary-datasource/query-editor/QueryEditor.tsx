import React, { useEffect, useState, type FC } from 'react';
import type { QueryEditorProps, SelectableValue } from '@grafana/data';
import { getTemplateSrv } from '@grafana/runtime';
import { Field, FieldSet, Select } from '@grafana/ui';
import type { PTSummaryDataSource } from '../PTSummaryDataSource';
import { toPTSummaryQuery, type DatasourceType } from '../PTSummary.types';
import { DATASOURCES } from './QueryEditor.constants';
import { Messages } from './QueryEditor.messages';

export const QueryEditor: FC<QueryEditorProps<PTSummaryDataSource>> = ({
  query,
  onChange,
}) => {
  const { queryType } = toPTSummaryQuery(query);
  const variablesOptions = getTemplateSrv()
    .getVariables()
    .map((variable) => ({
      value: variable.name,
      label: variable.label || undefined,
    }));
  const selectedOption = variablesOptions.find(
    (variable) => variable.value === queryType?.variableName
  );
  const selectedTypeOption = DATASOURCES.find(
    (datasource) => datasource.value === queryType?.type
  );
  const [selectedVariable, selectVariable] = useState<
    SelectableValue<string> | undefined
  >(selectedOption);
  const [type, setType] = useState<SelectableValue<DatasourceType>>(
    selectedTypeOption || DATASOURCES[0]
  );

  useEffect(() => {
    const newQueryType = { ...queryType };

    if (selectedVariable?.value) {
      newQueryType.variableName = selectedVariable.value;
    }

    if (type?.value) {
      newQueryType.type = type.value;
    }

    onChange({
      ...query,
      queryType: newQueryType,
    } as unknown as typeof query);
    // only the selections drive the query; query and onChange change with it
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedVariable, type]);

  return (
    <div className="gf-form">
      <FieldSet>
        <Field label={Messages.labels.field.datasourceType}>
          <Select
            options={DATASOURCES}
            value={selectedTypeOption}
            onChange={setType}
          />
        </Field>
        <Field label={Messages.labels.field.variableName}>
          <Select
            options={variablesOptions}
            value={selectedOption}
            onChange={selectVariable}
          />
        </Field>
      </FieldSet>
    </div>
  );
};
