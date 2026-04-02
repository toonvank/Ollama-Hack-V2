import { Input } from "@heroui/input";
import { useEffect, useState } from "react";

import { SearchIcon } from "./icons";

interface SearchInputProps {
  searchTerm: string;
  placeholder: string;
  setSearchTerm: (value: string) => void;
  handleSearch: (e: React.FormEvent) => void;
  autoSearchDelay?: number; // Auto-search delay (ms)
}

const SearchForm = ({
  placeholder,
  searchTerm,
  setSearchTerm,
  handleSearch,
  autoSearchDelay = 0, // No auto-search by default
}: SearchInputProps) => {
  const [localSearchTerm, setLocalSearchTerm] = useState(searchTerm);

  // When external searchTerm changes, update local searchTerm
  useEffect(() => {
    setLocalSearchTerm(searchTerm);
  }, [searchTerm]);

  // Handle search input change with auto-search support
  const handleSearchChange = (value: string) => {
    setLocalSearchTerm(value);
    setSearchTerm(value);

    // If auto-search delay is set, enable debounced auto-search
    if (autoSearchDelay > 0) {
      const timer = setTimeout(() => {
        const syntheticEvent = {
          preventDefault: () => {},
        } as React.FormEvent;

        handleSearch(syntheticEvent);
      }, autoSearchDelay);

      return () => clearTimeout(timer);
    }
  };

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
        onClear={() => {
          setLocalSearchTerm("");
          setSearchTerm("");
        }}
      />
    </form>
  );
};

export default SearchForm;
