import { createContext, ParentComponent, createSignal } from 'solid-js';
import type { AppContextType, CollectionCardData, FileData } from './library/types';

const AppContext = createContext<AppContextType>()

const ContextProvider: ParentComponent = (props) => {

  const [files, setFiles] = createSignal<Array<FileData>>([]);
  const [userCollections, setUserCollections] = createSignal<Array<CollectionCardData>>([]);
  const contextValue: AppContextType = {
    files: files,
    setFiles: setFiles,
    userCollections: userCollections,
    setUserCollections: setUserCollections,
  };

  return (
    <AppContext.Provider value={contextValue}>
      {props.children}
    </AppContext.Provider>
  );
}

export { AppContext, ContextProvider };
