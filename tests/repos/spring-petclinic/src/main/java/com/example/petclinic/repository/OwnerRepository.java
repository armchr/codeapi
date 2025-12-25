package com.example.petclinic.repository;

import com.example.petclinic.model.Owner;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.util.List;
import java.util.Optional;

/**
 * Repository for Owner entity with custom query methods.
 */
@Repository
public interface OwnerRepository extends JpaRepository<Owner, Long> {

    /**
     * Find owners by last name using LIKE query.
     */
    @Query("SELECT o FROM Owner o WHERE LOWER(o.lastName) LIKE LOWER(CONCAT('%', :lastName, '%'))")
    List<Owner> findByLastNameContaining(@Param("lastName") String lastName);

    /**
     * Find owner by email address.
     */
    Optional<Owner> findByEmail(String email);

    /**
     * Find owners by city.
     */
    List<Owner> findByCity(String city);

    /**
     * Find owners with their pets eagerly loaded.
     */
    @Query("SELECT DISTINCT o FROM Owner o LEFT JOIN FETCH o.pets")
    List<Owner> findAllWithPets();

    /**
     * Find owner by ID with pets eagerly loaded.
     */
    @Query("SELECT o FROM Owner o LEFT JOIN FETCH o.pets WHERE o.id = :id")
    Optional<Owner> findByIdWithPets(@Param("id") Long id);

    /**
     * Check if email already exists.
     */
    boolean existsByEmail(String email);

    /**
     * Count owners by city.
     */
    long countByCity(String city);
}
