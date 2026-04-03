import { Input } from "@nextui-org/input";
import { useCallback, useEffect, useRef, useState } from "react";

import { SearchIcon } from "./icons";

interface SearchInputProps {
  searchTerm: string;
  placeholder: string;
  setSearchTerm: (value: string) => void;
  handleSearch: (e: React.FormEvent) => void;
  autoSearchDelay?: number;
}

const SearchForm = ({
  placeholder,
  searchTerm,
  setSearchTerm,
  handleSearch,
  autoSearchDelay = 0,
}: SearchInputProps) => {
  const [localSearchTerm, setLocalSearchTerm] = useState(searchTerm);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Store latest callbacks in refs to avoid dependency churn
  const handleSearchRef = useRef(handleSearch);
  const setSearchTermRef = useRef(setSearchTerm);

  handleSearchRef.current = handleSearch;
  setSearchTermRef.current = setSearchTerm;

  // Sync from external searchTerm changes (e.g. URL state reset)
  useEffect(() => {
    setLocalSearchTerm(searchTerm);
  }, [searchTerm]);

  // Debounced auto-search
  useEffect(() => {
    if (autoSearchDelay <= 0) return;
    if (localSearchTerm === searchTerm) return;

    timerRef.current = setTimeout(() => {
      setSearchTermRef.current(localSearchTerm);
      handleSearchRef.current({ preventDefault: () => {} } as React.FormEvent);
    }, autoSearchDelay);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [localSearchTerm, autoSearchDelay, searchTerm]);

  const handleSearchChange = useCallback(
    (value: string) => {
      setLocalSearchTerm(value);

      // If no auto search delay, update external immediately
      if (autoSearchDelay <= 0) {
        setSearchTermRef.current(value);
      }
    },
    [autoSearchDelay],
  );

  const handleClear = useCallback(() => {
    setLocalSearchTerm("");
    setSearchTermRef.current("");
  }, []);

  return (
    <form className="w-full sm:w-auto flex" onSubmit={handleSearch}>
      <Input
        isClearable
        className="mr-2"
        color="primary"
        placeholder={placeholder}
        startContent={
          <SearchIcon className="text-black/50 mb-0.5 dark:text-white/90 text-slate-400 pointer-events-none flex-shrink-0" />
        }
        value={localSearchTerm}
        variant="bordered"
        onChange={(e) => handleSearchChange(e.target.value)}
        onClear={handleClear}
      />
    </form>
  );
};

export default SearchForm;
