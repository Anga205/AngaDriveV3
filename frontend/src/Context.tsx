import { createContext, ParentComponent, createSignal } from 'solid-js';
import type { AppContextType, FileData, KnownCollectionCards, KnownCollections } from './library/types';

const AppContext = createContext<AppContextType>()

const ContextProvider: ParentComponent = (props) => {

  const [files, setFiles] = createSignal<Array<FileData>>([]);
  const [userCollections, setUserCollections] = createSignal<Set<string>>(new Set());
  const [knownCollections, setKnownCollections] = createSignal<KnownCollections>({});
  const [knownCollectionCards, setKnownCollectionCards] = createSignal<KnownCollectionCards>({});
  const contextValue: AppContextType = {
    files: files,
    setFiles: setFiles,
    userCollections: userCollections,
    setUserCollections: setUserCollections,
    knownCollections: knownCollections,
    setKnownCollections: setKnownCollections,
    knownCollectionCards: knownCollectionCards,
    setKnownCollectionCards: setKnownCollectionCards,
  };

  return (
    <AppContext.Provider value={contextValue}>
      {props.children}
    </AppContext.Provider>
  );
}

export { AppContext, ContextProvider };
