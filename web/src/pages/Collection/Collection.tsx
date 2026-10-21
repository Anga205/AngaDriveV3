import { Component } from "solid-js";
import { useIsMobile } from "@/library/useIsMobile";
import CollectionPageDesktop from "./Desktop/CollectionPageDesktop";
import CollectionPageMobile from "./Mobile/CollectionPageMobile";

const CollectionPage: Component = () => {
    const isMobile = useIsMobile();

    return (
        <>
            {isMobile() ? <CollectionPageMobile /> : <CollectionPageDesktop />}
        </>
    );
}

export default CollectionPage;