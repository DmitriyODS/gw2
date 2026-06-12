-- +goose Up
-- Локация пользователя для погодных механик Грувика (groovesvc).
-- Координаты задаёт сам пользователь (геолокация браузера или поиск города),
-- city — человекочитаемая подпись из геокодинга, может отсутствовать.
CREATE TABLE public.user_locations (
    user_id    integer NOT NULL,
    latitude   double precision NOT NULL,
    longitude  double precision NOT NULL,
    city       character varying(120),
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.user_locations
    ADD CONSTRAINT user_locations_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY public.user_locations
    ADD CONSTRAINT user_locations_user_id_fkey FOREIGN KEY (user_id)
    REFERENCES public.users(id) ON DELETE CASCADE;

-- +goose Down
DROP TABLE IF EXISTS public.user_locations;
