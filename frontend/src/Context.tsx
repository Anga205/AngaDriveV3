import { createContext, ParentComponent, createSignal } from 'solid-js';
import type { AppContextType, FileData } from './library/types';

const AppContext = createContext<AppContextType>()

const ContextProvider: ParentComponent = (props) => {

  const [files, setFiles] = createSignal<Array<FileData>>([]);
  const contextValue: AppContextType = {
    files: files,
    setFiles: setFiles,
  };

  return (
    <AppContext.Provider value={contextValue}>
      {props.children}
    </AppContext.Provider>
  );
}

export { AppContext, ContextProvider };
