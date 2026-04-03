import React, { useState } from "react";
import { Button } from "@heroui/button";
import { Input } from "@heroui/input";
import { Textarea } from "@heroui/input";
import { Tabs, Tab } from "@heroui/tabs";
import { RadioGroup, Radio } from "@heroui/radio";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@heroui/modal";
import { addToast } from "@heroui/toast";
import { Form } from "@heroui/form";

import { endpointApi } from "@/api";
import { EndpointCreate } from "@/types";

interface CreateEndpointModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const CreateEndpointModal: React.FC<CreateEndpointModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const [selectedTab, setSelectedTab] = useState("single");
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Create endpoint form state
  const [formData, setFormData] = useState<EndpointCreate>({
    url: "",
    name: "",
    endpoint_type: "ollama",
    api_key: "",
  });

  // Batch create endpoint form state
  const [urls, setUrls] = useState("");

  // Handle form input change
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;

    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // Handle batch create textarea change
  const handleTextChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setUrls(e.target.value);
  };

  // Handle create endpoint form submit
  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      // Form validation
      if (!formData.url) {
        throw new Error("URL cannot be empty");
      }

      // Ensure URL format is correct
      let url = formData.url;

      if (!url.startsWith("http://") && !url.startsWith("https://")) {
        url = `http://${url}`;
      }

      // If name is empty, use URL as name
      let name = formData.name;

      if (!name) {
        name = new URL(url).hostname;
      }

      // Submit create request
      await endpointApi.createEndpoint({
        url,
        name,
        endpoint_type: formData.endpoint_type || "ollama",
        api_key: formData.api_key || undefined,
      });

      // Success, close modal and refresh list
      handleClose();
      onSuccess();

      // Reset form
      setFormData({
        url: "",
        name: "",
        endpoint_type: "ollama",
        api_key: "",
      });
      setSelectedTab("single");
    } catch (err) {
      addToast({
        title: "Failed to create endpoint",
        description: (err as Error)?.message || "Please try again",
        color: "danger",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle batch create form submit
  const handleBatchSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      // Split URLs and filter empty lines
      const urlLines = urls
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line);

      if (urlLines.length === 0) {
        throw new Error("Please enter at least one valid endpoint URL");
      }

      // Handle batch create
      const endpoints: EndpointCreate[] = urlLines.map((url) => {
        // Ensure URL format is correct
        let processedUrl = url;

        if (!url.startsWith("http://") && !url.startsWith("https://")) {
          processedUrl = `http://${url}`;
        }

        // Auto-generate name
        let name: string;

        try {
          name = new URL(processedUrl).hostname;
        } catch {
          name = processedUrl;
        }

        return {
          url: processedUrl,
          name: name,
        };
      });

      // Submit batch create request
      await endpointApi.batchCreateEndpoints({
        endpoints,
      });

      // Set success message and clear form
      addToast({
        title: `Successfully created ${endpoints.length} endpoints`,
        color: "success",
      });
      setUrls("");
      handleClose();
      onSuccess();
    } catch (err) {
      addToast({
        title: "Failed to batch create endpoints",
        description: (err as Error)?.message || "Please try again",
        color: "danger",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  // Close modal and reset form
  const handleClose = () => {
    if (!isSubmitting) {
      onClose();
      setSelectedTab("single");
      setFormData({
        url: "",
        name: "",
        endpoint_type: "ollama",
        api_key: "",
      });
      setUrls("");
    }
  };

  return (
    <Modal isOpen={isOpen} placement="center" onClose={handleClose}>
      <ModalContent>
        {() => (
          <Form
            className="w-full"
            id="create-endpoint-form"
            onSubmit={
              selectedTab === "single" ? handleCreateSubmit : handleBatchSubmit
            }
          >
            <ModalHeader>Create New Endpoint</ModalHeader>
            <ModalBody className="w-full">
              <Tabs
                classNames={{
                  tabList: "mb-4",
                }}
                selectedKey={selectedTab}
                onSelectionChange={setSelectedTab as (key: string) => void}
              >
                <Tab key="single" title="Single">
                  <div className="space-y-4">
                    <div className="mb-4">
                      <Input
                        isRequired
                        className="w-full"
                        description="Enter the full Ollama service URL, including protocol and port"
                        id="url"
                        label="Endpoint URL"
                        name="url"
                        placeholder="e.g. http://localhost:11434"
                        value={formData.url}
                        onChange={handleInputChange}
                      />
                    </div>

                    <div className="mb-6">
                      <Input
                        className="w-full"
                        description="Optional. If left empty, the URL hostname will be used"
                        id="name"
                        label="Endpoint Name"
                        name="name"
                        placeholder="Give this endpoint a name"
                        value={formData.name}
                        onChange={handleInputChange}
                      />
                    </div>

                    <div className="mb-4">
                      <RadioGroup
                        label="Endpoint Type"
                        description="Select the type of API endpoint"
                        value={formData.endpoint_type}
                        onValueChange={(value) =>
                          setFormData((prev) => ({
                            ...prev,
                            endpoint_type: value,
                          }))
                        }
                      >
                        <Radio value="ollama">
                          Ollama (Native API)
                        </Radio>
                        <Radio value="openai">
                          OpenAI Compatible (e.g., /v1/models)
                        </Radio>
                      </RadioGroup>
                    </div>

                    {formData.endpoint_type === "openai" && (
                      <div className="mb-4">
                        <Input
                          className="w-full"
                          description="API key for authentication (e.g., sk-...)"
                          id="api_key"
                          label="API Key"
                          name="api_key"
                          placeholder="sk-..."
                          type="password"
                          value={formData.api_key}
                          onChange={handleInputChange}
                        />
                      </div>
                    )}
                  </div>
                </Tab>
                <Tab key="batch" title="Batch Create">
                  <div className="mb-6">
                    <Textarea
                      isRequired
                      className="w-full min-h-[200px]"
                      description="Enter one URL per line. If no protocol is specified, http:// will be used by default. Hostnames will be used as endpoint names automatically."
                      id="urls"
                      label="Endpoint URLs"
                      placeholder={`Enter one URL per line, e.g.: \nhttp://localhost:11434 \n192.168.1.100:11434 \nollama-server-2:11434`}
                      value={urls}
                      onChange={handleTextChange}
                    />
                  </div>
                </Tab>
              </Tabs>
            </ModalBody>
            <ModalFooter className="w-full">
              <Button
                disabled={isSubmitting}
                variant="light"
                onPress={handleClose}
              >
                Cancel
              </Button>
              {selectedTab === "single" ? (
                <Button
                  color="primary"
                  disabled={isSubmitting}
                  isLoading={isSubmitting}
                  type="submit"
                >
                  Create
                </Button>
              ) : (
                <Button
                  color="primary"
                  disabled={isSubmitting}
                  isLoading={isSubmitting}
                  type="submit"
                >
                  {isSubmitting ? (
                    <>
                      <span className="ml-2">Creating...</span>
                    </>
                  ) : (
                    "Batch Create"
                  )}
                </Button>
              )}
            </ModalFooter>
          </Form>
        )}
      </ModalContent>
    </Modal>
  );
};

export default CreateEndpointModal;
