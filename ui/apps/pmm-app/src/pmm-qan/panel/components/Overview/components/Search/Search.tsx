import type { FC} from 'react';
import React, { useEffect, useMemo, useRef } from 'react';
import { Field, Form } from 'react-final-form';
import { Button, Icon, Input, useTheme } from '@grafana/ui';
import { debounce } from 'lodash';
import { Messages } from 'pmm-qan/panel/QueryAnalytics.messages';
import { getStyles } from './Search.styles';
import type { SearchProps, SearchValues } from './Search.types';

const SEARCH_DEBOUNCE_MS = 300;

export const Search: FC<SearchProps> = ({
  dataTestId,
  initialValue,
  handleSearch,
}) => {
  const theme = useTheme();
  const styles = getStyles(theme);
  const {
    search: { placeholder },
  } = Messages;
  const handleSearchRef = useRef(handleSearch);

  handleSearchRef.current = handleSearch;

  const debouncedSearch = useMemo(
    () =>
      debounce((search: string) => {
        handleSearchRef.current({ search });
      }, SEARCH_DEBOUNCE_MS),
    []
  );

  useEffect(
    () => () => {
      debouncedSearch.cancel();
    },
    [debouncedSearch]
  );

  const onSubmit = (values: SearchValues) => {
    debouncedSearch.cancel();
    handleSearch(values);
  };

  return (
    <Form
      onSubmit={onSubmit}
      initialValues={{ search: initialValue }}
      render={({ handleSubmit }) => (
        <form
          onSubmit={handleSubmit}
          className={styles.searchWrapper}
          data-testid={dataTestId}
        >
          <Field
            name="search"
            render={({ input }) => (
              <Input
                {...input}
                placeholder={placeholder}
                className={styles.searchInput}
                onChange={(event) => {
                  input.onChange(event);
                  debouncedSearch(event.currentTarget.value);
                }}
              />
            )}
          />
          <Button type="submit" className={styles.searchButton}>
            <Icon name="search" />
          </Button>
        </form>
      )}
    />
  );
};
