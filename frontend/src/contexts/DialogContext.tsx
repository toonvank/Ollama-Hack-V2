import React, { createContext, useState, useContext, ReactNode } from "react";

import DialogModal, { DialogType } from "@/components/DialogModal";

interface DialogContextType {
  confirm: (message: string, onConfirm: () => void, title?: string) => void;
}

const DialogContext = createContext<DialogContextType | undefined>(undefined);

export const useDialog = (): DialogContextType => {
  const context = useContext(DialogContext);

  if (!context) {
    throw new Error("useDialog must be used inside a DialogProvider");
  }

  return context;
};

interface DialogProviderProps {
  children: ReactNode;
}

export const DialogProvider: React.FC<DialogProviderProps> = ({ children }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [dialogContent, setDialogContent] = useState({
    title: "",
    message: "",
    type: "error" as DialogType,
    onConfirm: () => {},
  });

  const confirm = (message: string, onConfirm: () => void, title?: string) => {
    setDialogContent({
      title: title || "Confirm Action",
      message,
      type: "confirm",
      onConfirm,
    });
    setIsOpen(true);
  };

  const handleClose = () => {
    setIsOpen(false);
  };

  return (
    <DialogContext.Provider value={{ confirm }}>
      {children}
      <DialogModal
        isOpen={isOpen}
        message={dialogContent.message}
        title={dialogContent.title}
        type={dialogContent.type}
        onClose={handleClose}
        onConfirm={dialogContent.onConfirm}
      />
    </DialogContext.Provider>
  );
};

export default DialogProvider;
