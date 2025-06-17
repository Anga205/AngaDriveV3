import { createContext, ParentComponent, createSignal } from 'solid-js';
import { FileData } from './types/types';



type AppContextType = {
  files: () => Array<FileData>;
  setFiles: (value: Array<FileData> | ((prev: Array<FileData>) => Array<FileData>)) => void;
};

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
