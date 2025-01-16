import {
  createContext,
  FC,
  RefObject,
  PropsWithChildren,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import { Location, useNavigate } from 'react-router-dom';
import { constructUrlFromLocation } from 'utils/url';

export interface Message {
  type: string;
  data: any;
}

interface Ctx {
  frameRef: RefObject<HTMLIFrameElement> | undefined;
  isReady: boolean;
  messages: Message[];
  sendMessage: (msg: Message) => void;
}

const MessagesContext = createContext<Ctx>({
  frameRef: undefined,
  isReady: false,
  messages: [],
  sendMessage: () => null,
});

export const useMessenger = () => useContext(MessagesContext);

export const MessagesProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigate = useNavigate();
  const [isReady, setIsReady] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const [messages, setMessages] = useState<Message[]>([]);

  const getUrl = (location: Location) => {
    if (location.pathname.includes('/pmm-ui')) {
      location.pathname = location.pathname.replace('/pmm-ui', '');
      return constructUrlFromLocation(location);
    } else if (!location.pathname.includes('/graph')) {
      return '/graph' + constructUrlFromLocation(location);
    }

    return constructUrlFromLocation(location);
  };

  const handleLocationChange = (location: Location) => {
    console.log('LOCATION_CHANGE', location);
    console.log({
      pmm: window.location.pathname,
      grafana: location.pathname,
    });

    if (
      !window.location.pathname.includes('/graph') &&
      !location.pathname.includes('query-analytics')
    ) {
      return;
    }

    if (location.pathname.includes('pmm-qan/pmm-query-analytics')) {
      location.pathname = '/pmm-ui/query-analytics';
    }

    console.log({
      pmm: window.location.pathname,
      grafana: location.pathname,
    });

    const url = getUrl(location);

    console.log('url', url);

    navigate(url);
  };

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMessageReceived = (e: any) => {
      if (e.data) {
        const msg = e.data as Message;
        setMessages((messages) => [...messages, msg]);
      }

      if (e.data && e.data.type === 'LOCATION_CHANGE') {
        handleLocationChange(e.data.data.location);
      }

      if (e.data && e.data.type === 'MESSENGER_READY') {
        setIsReady(true);
      }
    };

    console.log('pmm', 'messager');
    window.addEventListener('message', onMessageReceived);

    return () => {
      window.removeEventListener('message', onMessageReceived);
    };
  }, []);

  const sendMessage = (msg: object) => {
    frameRef.current?.contentWindow?.postMessage(msg, '*');
  };

  return (
    <MessagesContext.Provider
      value={{
        frameRef,
        isReady,
        messages,
        sendMessage,
      }}
    >
      {children}
    </MessagesContext.Provider>
  );
};
