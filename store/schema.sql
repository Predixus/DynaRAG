--
-- PostgreSQL database dump
--

-- Dumped from database version 16.4 (Debian 16.4-1.pgdg120+2)
-- Dumped by pg_dump version 17.2 (Ubuntu 17.2-1.pgdg22.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: embedding_model; Type: TYPE; Schema: public; Owner: admin
--

CREATE TYPE public.embedding_model AS ENUM (
    'all-MiniLM-L6-v2',
    'all-mpnet-base-v2',
    'multi-CAUTION-MiniLM-L6-cos-v1'
);


ALTER TYPE public.embedding_model OWNER TO admin;

--
-- Name: check_document_embeddings(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.check_document_embeddings() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- If no embeddings exist for this document
    IF NOT EXISTS (
        SELECT 1 
        FROM embeddings 
        WHERE document_id = OLD.document_id
    ) THEN
        -- Get the document's user_id before deleting it
        DELETE FROM documents WHERE id = OLD.document_id;
    END IF;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.check_document_embeddings() OWNER TO admin;

--
-- Name: increment_api_usage(bigint); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.increment_api_usage(user_id_param bigint) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE api_usage
    SET 
        request_count = request_count + 1,
        last_request_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    WHERE user_id = user_id_param;
END;
$$;


ALTER FUNCTION public.increment_api_usage(user_id_param bigint) OWNER TO admin;

--
-- Name: initialize_api_usage(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.initialize_api_usage() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO api_usage (user_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.initialize_api_usage() OWNER TO admin;

--
-- Name: reset_user_chunk_size(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.reset_user_chunk_size() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- After a document is deleted, check if the user has any remaining documents
    IF NOT EXISTS (
        SELECT 1 
        FROM documents 
        WHERE user_id = OLD.user_id
    ) THEN
        -- If no documents remain, reset the user's total_chunk_size to 0
        UPDATE users 
        SET 
            total_chunk_size = 0,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.user_id;
    ELSE
        -- If documents remain, recalculate the total from existing documents
        UPDATE users u
        SET 
            total_chunk_size = COALESCE((
                SELECT SUM(e.chunk_size)
                FROM documents d
                JOIN embeddings e ON d.id = e.document_id
                WHERE d.user_id = OLD.user_id
            ), 0),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.user_id;
    END IF;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.reset_user_chunk_size() OWNER TO admin;

--
-- Name: update_chunk_sizes(); Type: FUNCTION; Schema: public; Owner: admin
--

CREATE FUNCTION public.update_chunk_sizes() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- First update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = total_chunk_size + NEW.chunk_size,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.document_id;
        
        -- Then update the users total_chunk_size in a single step
        UPDATE users 
        SET 
            total_chunk_size = total_chunk_size + NEW.chunk_size,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = (
            SELECT user_id 
            FROM documents 
            WHERE id = NEW.document_id
        );
        
    ELSIF TG_OP = 'DELETE' THEN
        -- First update the documents total_chunk_size
        UPDATE documents 
        SET 
            total_chunk_size = GREATEST(0, total_chunk_size - OLD.chunk_size),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = OLD.document_id;
        
        -- Then update the users total_chunk_size in a single step
        UPDATE users 
        SET 
            total_chunk_size = GREATEST(0, total_chunk_size - OLD.chunk_size),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = (
            SELECT user_id 
            FROM documents 
            WHERE id = OLD.document_id
        );
    END IF;
    
    RETURN NULL;
END;
$$;


ALTER FUNCTION public.update_chunk_sizes() OWNER TO admin;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: api_usage; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.api_usage (
    id bigint NOT NULL,
    user_id bigint,
    request_count bigint DEFAULT 0,
    last_request_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.api_usage OWNER TO admin;

--
-- Name: api_usage_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.api_usage_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.api_usage_id_seq OWNER TO admin;

--
-- Name: api_usage_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.api_usage_id_seq OWNED BY public.api_usage.id;


--
-- Name: documents; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.documents (
    id bigint NOT NULL,
    user_id bigint,
    file_path text NOT NULL,
    total_chunk_size bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.documents OWNER TO admin;

--
-- Name: documents_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

-- a tsvector column for full-text search in the documents table

ALTER TABLE public.documents
ADD COLUMN text_searchable_column tsvector;


-- just converting file_path for now

UPDATE public.documents
SET text_searchable_column = to_tsvector('english', file_path);


CREATE SEQUENCE public.documents_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.documents_id_seq OWNER TO admin;

--
-- Name: documents_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.documents_id_seq OWNED BY public.documents.id;


--
-- Name: embeddings; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.embeddings (
    id bigint NOT NULL,
    document_id bigint,
    model_name public.embedding_model NOT NULL,
    embedding public.vector(384) NOT NULL,
    chunk_text text NOT NULL,
    chunk_size integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata_hash text,
    embedding_text text
)
WITH (toast_tuple_target='128');


ALTER TABLE public.embeddings OWNER TO admin;

--
-- Name: embeddings_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.embeddings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.embeddings_id_seq OWNER TO admin;

--
-- Name: embeddings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.embeddings_id_seq OWNED BY public.embeddings.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO admin;

--
-- Name: users; Type: TABLE; Schema: public; Owner: admin
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    user_id text NOT NULL,
    total_chunk_size bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.users OWNER TO admin;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: admin
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO admin;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: admin
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: api_usage id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.api_usage ALTER COLUMN id SET DEFAULT nextval('public.api_usage_id_seq'::regclass);


--
-- Name: documents id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.documents ALTER COLUMN id SET DEFAULT nextval('public.documents_id_seq'::regclass);


--
-- Name: embeddings id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.embeddings ALTER COLUMN id SET DEFAULT nextval('public.embeddings_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: api_usage api_usage_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.api_usage
    ADD CONSTRAINT api_usage_pkey PRIMARY KEY (id);


--
-- Name: api_usage api_usage_user_id_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.api_usage
    ADD CONSTRAINT api_usage_user_id_key UNIQUE (user_id);


--
-- Name: documents documents_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_pkey PRIMARY KEY (id);


--
-- Name: documents documents_user_id_file_path_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_user_id_file_path_key UNIQUE (user_id, file_path);


--
-- Name: embeddings embeddings_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_user_id_key; Type: CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_user_id_key UNIQUE (user_id);


--
-- Name: api_usage_user_id_idx; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX api_usage_user_id_idx ON public.api_usage USING btree (user_id);


--
-- Name: embeddings_chunk_text_idx; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX embeddings_chunk_text_idx ON public.embeddings USING gin (chunk_text public.gin_trgm_ops);


--
-- Name: embeddings_embedding_idx; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX embeddings_embedding_idx ON public.embeddings USING ivfflat (embedding public.vector_cosine_ops);


--
-- Name: embeddings_metadata_hash_idx; Type: INDEX; Schema: public; Owner: admin
--

CREATE INDEX embeddings_metadata_hash_idx ON public.embeddings USING btree (metadata_hash);


--
-- Name: users create_api_usage_record; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER create_api_usage_record AFTER INSERT ON public.users FOR EACH ROW EXECUTE FUNCTION public.initialize_api_usage();


--
-- Name: embeddings delete_empty_documents; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER delete_empty_documents AFTER DELETE ON public.embeddings FOR EACH ROW EXECUTE FUNCTION public.check_document_embeddings();


--
-- Name: documents reset_user_chunk_size_trigger; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER reset_user_chunk_size_trigger AFTER DELETE ON public.documents FOR EACH ROW EXECUTE FUNCTION public.reset_user_chunk_size();


--
-- Name: embeddings update_chunk_sizes_trigger; Type: TRIGGER; Schema: public; Owner: admin
--

CREATE TRIGGER update_chunk_sizes_trigger AFTER INSERT OR DELETE ON public.embeddings FOR EACH ROW EXECUTE FUNCTION public.update_chunk_sizes();


--
-- Name: api_usage api_usage_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.api_usage
    ADD CONSTRAINT api_usage_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: documents documents_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: admin
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

-- creating a gin index to speed up text search operations
CREATE INDEX idx_text_searchable_column
ON public.documents USING gin (text_searchable_column);


-- trigger to automatically update text_searchable_column
   CREATE TRIGGER documents_tsvector_update
   BEFORE INSERT OR UPDATE ON public.documents
   FOR EACH ROW
   EXECUTE FUNCTION tsvector_update_trigger(
       text_searchable_column,
       'pg_catalog.english',
       file_path
   );


